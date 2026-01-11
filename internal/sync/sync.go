package sync

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/sym01/htmlsanitizer"

	"github.com/jdholdren/seymour/internal/seymour"
)

// Represents a response from an RSS feed fetch.
type rssFeedResp struct {
	XMLName xml.Name `xml:"rss"`
	Channel []struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Link        string `xml:"link"`
		Items       []struct {
			Title       string   `xml:"title"`
			Links       []string `xml:"link"`
			GUID        string   `xml:"guid"`
			Description string   `xml:"description"`
			PubDate     string   `xml:"pubDate"`
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
			// Parse the links to the post
			var nonEmptyLink string
			for _, link := range item.Links {
				if link == "" {
					continue
				}

				nonEmptyLink = link
			}

			// Parse the publish date
			var publishedAt *time.Time
			if parsedTime, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				publishedAt = &parsedTime
			}
			if parsedTime, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				publishedAt = &parsedTime
			}

			entries = append(entries, seymour.FeedEntry{
				FeedID:      feedID,
				GUID:        item.GUID,
				Title:       sanitize(item.Title),
				Description: sanitize(item.Description),
				Link:        nonEmptyLink,
				PublishTime: publishedAt,
			})
		}
	}

	// Only the fields being updated:
	return seymour.Feed{
		ID:          feedID,
		Title:       &feedResp.Channel[0].Title,
		Description: &feedResp.Channel[0].Description,
	}, entries, nil
}

// Removes all html tags from the string, usually a description.
//
// Also limits the length of the string so there's not a massive chunk of text being output.
func sanitize(s string) string {
	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizer.AllowList = nil // Remove anything HTML

	// Remove any extra HTML
	s, err := sanitizer.SanitizeString(s)
	if err != nil {
		return ""
	}

	// Unescape any HTML codes into the real thing for display
	s = html.UnescapeString(s)

	// Remove whitespace and newlines
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")

	// Keep the length under 2048: some feeds put the whole post in there.
	if len(s) > 2048 {
		s = s[:2048]
	}

	return s
}
