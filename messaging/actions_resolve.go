package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ringclaw/ringclaw/ringcentral"
)

// bestDirectoryMatch finds the best matching directory entry:
// exact match first, then shortest fuzzy match.
func bestDirectoryMatch(records []ringcentral.DirectoryEntry, name string) *ringcentral.DirectoryEntry {
	// Pass 1: exact match (case-insensitive)
	for i := range records {
		e := &records[i]
		fullName := strings.TrimSpace(e.FirstName + " " + e.LastName)
		if exactMatch(fullName, name) || exactMatch(e.Email, name) {
			return e
		}
	}
	// Pass 2: fuzzy match — prefer the shortest full name (closest to input)
	var best *ringcentral.DirectoryEntry
	bestLen := int(^uint(0) >> 1) // max int
	for i := range records {
		e := &records[i]
		fullName := strings.TrimSpace(e.FirstName + " " + e.LastName)
		if fuzzyMatch(fullName, name) || fuzzyMatch(e.Email, name) {
			if len(fullName) < bestLen {
				best = e
				bestLen = len(fullName)
			}
		}
	}
	return best
}

// resolveNameToChatID resolves a person name to a Direct chat ID via directory search.
func resolveNameToChatID(ctx context.Context, client *ringcentral.Client, name string) (string, error) {
	result, err := client.SearchDirectory(ctx, name)
	if err != nil {
		return "", fmt.Errorf("directory search: %w", err)
	}
	if len(result.Records) == 0 {
		return "", fmt.Errorf("no person found matching '%s'", name)
	}

	best := bestDirectoryMatch(result.Records, name)
	if best == nil {
		return "", fmt.Errorf("no person matched '%s' (got %d results)", name, len(result.Records))
	}

	fullName := strings.TrimSpace(best.FirstName + " " + best.LastName)
	slog.Info("action: resolved person", "name", name, "match", fullName, "id", best.ID)

	chat, err := client.CreateConversation(ctx, []string{best.ID})
	if err != nil {
		return "", fmt.Errorf("create conversation with %s: %w", fullName, err)
	}
	return chat.ID, nil
}

// resolveNameToPersonID resolves a person name to a person ID via directory search.
func resolveNameToPersonID(ctx context.Context, client *ringcentral.Client, name string) (string, error) {
	result, err := client.SearchDirectory(ctx, name)
	if err != nil {
		return "", fmt.Errorf("directory search: %w", err)
	}
	if len(result.Records) == 0 {
		return "", fmt.Errorf("no person found matching '%s'", name)
	}

	best := bestDirectoryMatch(result.Records, name)
	if best == nil {
		return "", fmt.Errorf("no person matched '%s'", name)
	}

	fullName := strings.TrimSpace(best.FirstName + " " + best.LastName)
	slog.Info("action: resolved assignee", "name", name, "match", fullName, "id", best.ID)
	return best.ID, nil
}

func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// resolveChatParam resolves a chatid param: numeric IDs pass through,
// names are resolved via directory search + conversation creation.
func resolveChatParam(ctx context.Context, client *ringcentral.Client, raw string) (string, error) {
	id := extractChatID(raw)
	if isNumericID(id) {
		return id, nil
	}
	return resolveNameToChatID(ctx, client, id)
}

// resolveAssigneeParam resolves an assignee param: numeric IDs pass through,
// names are resolved via directory search.
func resolveAssigneeParam(ctx context.Context, client *ringcentral.Client, raw string) (string, error) {
	id := extractChatID(raw)
	if isNumericID(id) {
		return id, nil
	}
	return resolveNameToPersonID(ctx, client, id)
}

// selectCardClient picks the right client for adaptive card creation.
func selectCardClient(replyClient, actionClient *ringcentral.Client, targetChat string) *ringcentral.Client {
	if replyClient != nil && replyClient.IsBot() && replyClient.IsBotDM(targetChat) {
		return replyClient
	}
	if actionClient != nil {
		return actionClient
	}
	return replyClient
}
