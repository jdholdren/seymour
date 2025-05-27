package agg

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
)

type (
	// Aggregator provides the functionality for syncing feeds and aggregating their entries.
	Aggregator struct {
		repo Repo
	}

	// Feed represents an RSS feed's details.
	Feed struct {
		ID           string     `db:"id"`
		Title        *string    `db:"title"`
		URL          string     `db:"url"`
		Description  *string    `db:"description"`
		LastSyncedAt *time.Time `db:"last_synced_at"`
		CreatedAt    time.Time  `db:"created_at"`
		UpdatedAt    time.Time  `db:"updated_at"`
	}

	// Entry represents a unique entry in an RSS feed.
	Entry struct {
		ID          string    `db:"id"`
		FeedID      string    `db:"feed_id"`
		GUID        string    `db:"guid"`
		Title       string    `db:"title"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
	}
)

var (
	ErrConflict = errors.New("resource already exists")
	ErrNotFound = errors.New("resource not found")
)

func NewAggregator(repo Repo) Aggregator {
	return Aggregator{
		repo: repo,
	}
}

func (a Aggregator) Feed(ctx context.Context, feedID string) (Feed, error) {
	feed, err := a.repo.feed(ctx, feedID)
	if errors.Is(err, ErrNotFound) {
		return Feed{}, seyerrs.E(err, http.StatusNotFound)
	}
	if err != nil {
		return Feed{}, seyerrs.E(err)
	}

	return feed, nil
}

func (a Aggregator) FeedByURL(ctx context.Context, feedURL string) (Feed, error) {
	feed, err := a.repo.feedByURL(ctx, feedURL)
	if errors.Is(err, ErrNotFound) {
		return Feed{}, seyerrs.E(err, http.StatusNotFound)
	}
	if err != nil {
		return Feed{}, seyerrs.E(err)
	}

	return feed, nil
}

func (a Aggregator) InsertFeed(ctx context.Context, feedURL string) (Feed, error) {
	// TODO: URL validation

	feed, err := a.repo.insertFeed(ctx, feedURL)
	if errors.Is(err, ErrConflict) {
		return Feed{}, seyerrs.E(err, http.StatusConflict)
	}
	if err != nil {
		return Feed{}, seyerrs.E(err)
	}

	return feed, nil
}

func (a Aggregator) RemoveFeed(ctx context.Context, feedID string) error {
	if err := a.repo.deleteFeed(ctx, feedID); err != nil {
		return seyerrs.E(err)
	}

	return nil
}

func (a Aggregator) AllFeeds(ctx context.Context) ([]Feed, error) {
	feeds, err := a.repo.allFeeds(ctx)
	if err != nil {
		return nil, seyerrs.E(err)
	}

	return feeds, nil
}

// Represents a response from an RSS feed fetch.
type rssFeedResp struct {
	XMLName xml.Name `xml:"rss"`
	Channel []struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Link        string `xml:"link"`
		Items       []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			GUID        string `xml:"guid"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

var syncClient = &http.Client{
	Timeout: time.Second * 3,
}

func (a Aggregator) SyncFeed(ctx context.Context, feedID string) error {
	feed, err := a.repo.feed(ctx, feedID)
	if err != nil {
		return fmt.Errorf("error fetching feed to sync: %w", err)
	}

	resp, err := syncClient.Get(feed.URL)
	if err != nil {
		return fmt.Errorf("error getting feed url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feedResp rssFeedResp
	if err := xml.NewDecoder(resp.Body).Decode(&feedResp); err != nil {
		return fmt.Errorf("error decoding feed: %s", err)
	}

	entries := []Entry{}
	for _, channel := range feedResp.Channel {
		for _, item := range channel.Items {
			entries = append(entries, Entry{
				FeedID:      feedID,
				GUID:        item.GUID,
				Title:       sanitize(item.Title),
				Description: sanitize(item.Description),
			})
		}
	}

	// Persist the new stuff
	if err := a.repo.insertEntries(ctx, entries); err != nil {
		return fmt.Errorf("error inserting entries: %s", err)
	}
	if err := a.repo.updateFeed(ctx, feedID, UpdateFeedArgs{
		Title:       feedResp.Channel[0].Title,
		Description: feedResp.Channel[0].Title,
		LastSynced:  time.Now(),
	}); err != nil {
		return fmt.Errorf("error updating feed title: %s", err)
	}

	return nil
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
