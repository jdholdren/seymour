package sync

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jdholdren/seymour/internal/seymour"
	"github.com/microcosm-cc/bluemonday"
)

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

func Feed(ctx context.Context, feedID, feedURL string) (seymour.Feed, []seymour.FeedEntry, error) {
	resp, err := syncClient.Get(feedURL)
	if err != nil {
		return seymour.Feed{}, nil, fmt.Errorf("error getting feed url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return seymour.Feed{}, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feedResp rssFeedResp
	if err := xml.NewDecoder(resp.Body).Decode(&feedResp); err != nil {
		return seymour.Feed{}, nil, fmt.Errorf("error decoding feed: %w", err)
	}

	entries := []seymour.FeedEntry{}
	for _, channel := range feedResp.Channel {
		for _, item := range channel.Items {
			entries = append(entries, seymour.FeedEntry{
				FeedID:      feedID,
				GUID:        item.GUID,
				Title:       sanitize(item.Title),
				Description: sanitize(item.Description),
			})
		}
	}

	// Only the fields being updated:
	return seymour.Feed{
		Title:       &feedResp.Channel[0].Title,
		Description: &feedResp.Channel[0].Description,
	}, entries, nil
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
