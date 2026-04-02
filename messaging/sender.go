package messaging

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ringclaw/ringclaw/internal/util"
	"github.com/ringclaw/ringclaw/ringcentral"
)

// SendTypingPlaceholder sends a "Thinking..." placeholder message and returns its post ID.
func SendTypingPlaceholder(ctx context.Context, client *ringcentral.Client, chatID string) (string, error) {
	post, err := client.SendPost(ctx, chatID, "Thinking...")
	if err != nil {
		return "", fmt.Errorf("send typing placeholder: %w", err)
	}
	slog.Info("sent typing placeholder", "component", "sender", "chatID", chatID, "postID", post.ID)
	return post.ID, nil
}

// UpdatePostText updates an existing post's text content.
func UpdatePostText(ctx context.Context, client *ringcentral.Client, chatID, postID, text string) error {
	_, err := client.UpdatePost(ctx, chatID, postID, text)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	slog.Info("updated post", "component", "sender", "postID", postID, "chatID", chatID, "text", util.Truncate(text, 50))
	return nil
}

// SendTextReply sends a text reply to a chat.
func SendTextReply(ctx context.Context, client *ringcentral.Client, chatID, text string) error {
	_, err := client.SendPost(ctx, chatID, text)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	slog.Info("sent reply", "component", "sender", "chatID", chatID, "text", util.Truncate(text, 50))
	return nil
}

// logSendError logs a send error if non-nil. Use instead of _ = SendTextReply(...).
func logSendError(err error) {
	if err != nil {
		slog.Error("failed to send reply", "component", "sender", "error", err)
	}
}


