package worker

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

//go:embed system_prompt.txt
var systemPrompt string

//go:embed user_criteria.txt
var userCriteria string

type claudeJudgement struct {
	FeedEntryID string `json:"feed_entry_id"`
	Approved    bool   `json:"approved"`
}

// Use a schema to constrain the output
var (
	outputSchema = map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"feed_entry_id": map[string]any{"type": "string"},
				"approved":      map[string]any{"type": "boolean"},
			},
			"required": []string{"feed_entry_id", "approved"},
		},
	}
	outputFormat = anthropic.BetaJSONSchemaOutputFormat(outputSchema)
)

// JudgeEntries fetches the entries in need of judgement for a user and then
// approves them based on criteria.
//
// If the user has a prompt, send it to Claude to be judged.
// If the user does not have a prompt, auto-approve all entries.
func (a activities) JudgeEntries(ctx context.Context, userID string) (judgements, error) {
	l := activity.GetLogger(ctx)

	// Need to limit this in case we pull too many results. The call to claude will take longer and
	// likely hit a limit.
	entries, err := a.repo.EntriesNeedingJudgement(ctx, userID, 20)
	if err != nil {
		return nil, fmt.Errorf("error finding needing judgement timeline entries: %s", err)
	}

	l.Info("judging entries", "user_id", userID, "count", len(entries))

	// If no entries to judge, return empty result
	if len(entries) == 0 {
		return nil, nil
	}

	// Get user's preferences
	user, err := a.repo.User(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	if user.Prompt == "" {
		// User has no prompt, auto-approve and return early
		j := make(judgements)
		for _, entry := range entries {
			j[entry.ID] = true
		}

		return j, nil
	}

	var (
		entryIDs []string
		// Easier lookup: claude will use feed entry id, we need to get it back to timeline entry id
		feedEntryToTimeline = make(map[string]string)
	)
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.FeedEntryID)
		feedEntryToTimeline[entry.FeedEntryID] = entry.ID
	}
	// Fetch the full feed entries and construct the user message to claude
	feedEntries, err := a.repo.Entries(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed entries: %w", err)
	}
	byts, _ := json.Marshal(feedEntries)
	userMessage := fmt.Sprintf(userCriteria, string(user.Prompt), string(byts))

	// The call to claude to get the json of judgements:
	claudeResp, err := a.claudeClient.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model: anthropic.ModelClaudeHaiku4_5,
		Betas: []anthropic.AnthropicBeta{
			"structured-outputs-2025-11-13",
		},
		MaxTokens:    1024,
		OutputFormat: outputFormat,
		System: []anthropic.BetaTextBlockParam{{
			Text: systemPrompt,
		}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(userMessage)),
		},
	})
	// Anthropic error that we can handle
	var claudeErr *anthropic.Error
	if errors.As(err, &claudeErr) && claudeErr.StatusCode == http.StatusTooManyRequests {
		return nil, temporal.NewApplicationError("rate limit hit", errTypeRateLimit, err)
	}
	if err != nil {
		return nil, temporal.NewApplicationError("claude error", errTypeInternal, err)
	}

	var claudeJson string
	for _, content := range claudeResp.Content {
		claudeJson += content.Text
	}
	var claudeJudgements []claudeJudgement
	if err := json.Unmarshal([]byte(claudeJson), &claudeJudgements); err != nil {
		return nil, fmt.Errorf("error unmarshaling claude json: %s", err)
	}

	j := make(judgements)
	for _, judgement := range claudeJudgements {
		j[feedEntryToTimeline[judgement.FeedEntryID]] = judgement.Approved
	}

	return j, nil
}
