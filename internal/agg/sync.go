package agg

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jdholdren/seymour/internal/agg/database"
	"github.com/jdholdren/seymour/internal/agg/model"
	"github.com/jdholdren/seymour/logger"
)

// Syncer manages subscriptions of all rss feeds managed by the database.
//
// It asks the subscriptions to periodically fetch from RSS feeds and then updates
// the feeds in the db as well as inserting the new entries.
type Syncer struct {
	repo database.Repo

	// All currently known subscriptions for the loop
	mu   sync.Mutex
	subs []*subscription
}

func NewSyncer(r database.Repo) *Syncer {
	return &Syncer{
		repo: r,
	}
}

// Run starts the syncer loop.
func (s *Syncer) Run(ctx context.Context) error {
	ctx = logger.Ctx(ctx, slog.String("component", "syncer"))

	// Subscriptions are initially the feeds in the database:
	feeds, err := s.repo.AllFeeds(ctx)
	if err != nil {
		return fmt.Errorf("error initializing with all feeds: %s", err)
	}
	for _, feed := range feeds {
		s.subs = append(s.subs, subscribe(feed.URL))
	}
	updates := mergeSubs(s.subs...)

	var (
		toInsert []model.Entry
	)
	for {
		select {
		case <-ctx.Done():
			// Stop each subscription and then exit
			for _, sub := range s.subs {
				sub.stop()
			}
			return ctx.Err()
		case ups := <-updates:
			// Batch entries for insertion
			toInsert = append(toInsert, ups...)
		}
	}
}

// Periodically fetches from an RSS feed and emits items as a stream.
type subscription struct {
	feedUrl string // Where this subscription is fetching from
	updates chan []model.Entry
	close   chan struct{}
}

func subscribe(feedUrl string) *subscription {
	sub := &subscription{
		feedUrl: feedUrl,
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
			// TODO: Make nonblocking with another goroutine
			entries, err := s.fetch()
			if err != nil {
				slog.Error("error fetching %s", "error", err)
				fetchErrors += 1
			} else {
				// Days since last incident: 0
				fetchErrors = 0
			}

			// Filter out any entries already seen:
			for _, e := range entries {
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

			// Set the next time to fetch with a backoff for number of errors (linear backoff):
			timeToFetch = time.After(time.Duration(15+fetchErrors) * time.Second)
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
				GUID:        item.GUID,
				Title:       item.Title,
				Description: item.Description,
			})
		}
	}
	return ret, nil
}

// Turns a number of update channels into one.
func mergeSubs(subs ...*subscription) chan []model.Entry {
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
