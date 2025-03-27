package agg

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"github.com/jdholdren/seymour/internal/agg/database"
	"github.com/jdholdren/seymour/internal/agg/model"
	"github.com/jdholdren/seymour/logger"
)

// Syncer manages subscriptions of all rss feeds managed by the database.
//
// It asks the subscriptions to periodically fetch from RSS feeds and then updates
// the feeds in the db as well as inserting the new entries.
type Syncer struct {
	repo  database.Repo
	addCh chan model.Feed
}

func NewSyncer(repo database.Repo) *Syncer {
	return &Syncer{
		repo:  repo,
		addCh: make(chan model.Feed, 50),
	}
}

// Run starts the syncer loop.
func (s *Syncer) Run(ctx context.Context, feeds []model.Feed) error {
	ctx = logger.Ctx(ctx, slog.String("component", "syncer"))

	var subs []*subscription
	for _, feed := range feeds {
		subs = append(subs, subscribe(feed))
	}
	updates := mergeSubs(subs...)

	for {
		select {
		case <-ctx.Done():
			// Stop each subscription and then exit
			for _, sub := range subs {
				sub.stop()
			}
			return ctx.Err()
		case ups := <-updates:
			if err := s.repo.InsertEntries(ctx, ups); err != nil {
				return err
			}
		case feed := <-s.addCh:
			// Stop all subscriptions and then restart them
			for _, sub := range subs {
				sub.stop()
			}
			// Clear out the current subscriptions
			subs = []*subscription{}
			feeds = append(feeds, feed)

			// Subscribe to all feeds again
			for _, feed := range feeds {
				// TODO: Separate out starting the routine?
				subs = append(subs, subscribe(feed))
			}

			updates = mergeSubs(subs...)
		}
	}
}

func (s *Syncer) AddFeed(feed model.Feed) {
	s.addCh <- feed
}

// Periodically fetches from an RSS feed and emits items as a stream.
type subscription struct {
	feedUrl string // Where this subscription is fetching from
	feedID  string
	updates chan []model.Entry
	close   chan struct{}
}

func subscribe(feed model.Feed) *subscription {
	sub := &subscription{
		feedUrl: feed.URL,
		feedID:  feed.ID,
		updates: make(chan []model.Entry),
		close:   make(chan struct{}),
	}
	go sub.loop()

	return sub
}

// Periodically fetches from an RSS feed and emits items as a stream through
// the updates channel.
func (s *subscription) loop() {
	var (
		// At first, sync immediately
		timeToFetch = time.After(0)
		// Channel that is set to emit when a fetch starts and emits when the fetch has completed
		fetchDone chan fetchResult
		// Set to nil until there's updates to send to the receiver
		updates chan []model.Entry
		// Buffer of entries that need to be sent to the receiver after a fetch occurs
		send []model.Entry
		// Entries that have already been seen by the subscription and don't need to be sent again.
		// Key is the post guid.
		seen = map[string]struct{}{}
		// Number of consecutive errors encountered during fetches
		fetchErrors int
	)
	for {
		select {
		case <-s.close:
			// Close the updates channel
			close(s.updates)
			// Exit the loop
			return
		case <-timeToFetch:
			// Enable the fetchDone case by initializing the channel
			fetchDone = make(chan fetchResult)
			// Disable starting another fetch
			timeToFetch = nil

			// Start another routine that will emit the result back
			go func() {
				entries, err := s.fetch()
				fetchDone <- fetchResult{entries: entries, err: err}
				close(fetchDone)
			}()
		case result := <-fetchDone:
			if result.err != nil {
				slog.Error("error fetching %s", "error", result.err)
				fetchErrors += 1
			} else {
				// Days since last incident: 0
				fetchErrors = 0
			}

			// Filter out any entries already seen:
			for _, e := range result.entries {
				if _, ok := seen[e.GUID]; ok {
					continue
				}
				seen[e.GUID] = struct{}{} // Mark is as seen

				send = append(send, e)
			}
			// If there are  pending entries to send, enable the updates channel:
			if len(send) > 0 {
				updates = s.updates
			}

			// Disable the case to receive results
			fetchDone = nil

			// Set the next time to fetch with a backoff for number of errors (linear backoff):
			timeToFetch = time.After(time.Duration(15+fetchErrors) * time.Minute)
		case updates <- send:
			// Empty out the buffer:
			send = nil
			// Turn off this case so this isn't immediately hit.
			// Will be turned on when there's more pending entries to send:
			updates = nil
		}
	}
}

func (s *subscription) stop() {
	s.close <- struct{}{}
}

// Represents a response from an RSS feed fetch.
type rssFeedResp struct {
	XMLName xml.Name `xml:"rss"`
	Channel []struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			GUID        string `xml:"guid"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

type fetchResult struct {
	entries []model.Entry
	err     error
	// TODO: Something with the metadata of the feed
}

// Goes to the url and grabs the RSS feed items.
func (s *subscription) fetch() ([]model.Entry, error) {
	// TODO: Maybe emit the feed itself for title updates?
	client := &http.Client{
		Timeout: time.Second * 3,
	}
	resp, err := client.Get(s.feedUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting feed url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feed rssFeedResp
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("error decoding feed: %s", err)
	}

	ret := []model.Entry{}
	for _, channel := range feed.Channel {
		for _, item := range channel.Items {
			ret = append(ret, model.Entry{
				FeedID:      s.feedID,
				GUID:        item.GUID,
				Title:       sanitize(item.Title),
				Description: sanitize(item.Description),
			})
		}
	}
	return ret, nil
}

var stripPolicy = bluemonday.StrictPolicy()

// Removes all html tags from the string, usually a description.
//
// Also limits the length of the string so there's not a massive chunk of text being output.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = stripPolicy.Sanitize(s)
	if len(s) > 2048 {
		s = s[:2048]
	}

	return s
}

// Turns a number of update channels into one.
func mergeSubs(subs ...*subscription) chan []model.Entry {
	if len(subs) == 0 {
		return nil // Never emits
	}
	// The channel to return
	out := make(chan []model.Entry)
	// Wait group to wait for all channels to be closed
	var wg sync.WaitGroup

	// Start a routine for each channel to read from that forwards to the
	// output channel.
	forward := func(c chan []model.Entry) {
		defer wg.Done()
		for f := range c {
			out <- f
		}
	}
	wg.Add(len(subs))
	for _, c := range subs {
		go forward(c.updates)
	}

	// Final goroutine to wait for all channels to be closed and then close
	// the output channel
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
