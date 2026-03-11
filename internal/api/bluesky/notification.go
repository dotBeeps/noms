package bluesky

import (
	"context"
	"time"

	bsky "github.com/bluesky-social/indigo/api/bsky"
)

// ListNotifications fetches the user's notifications with pagination.
// Returns notifications, the next cursor, and any error.
func (c *Client) ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	var out bsky.NotificationListNotifications_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.notification.listNotifications", map[string]any{
			"cursor": cursor,
			"limit":  int64(limit),
		}, &out)
	})
	if err != nil {
		return nil, "", wrapErr("ListNotifications", err)
	}

	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return out.Notifications, nextCursor, nil
}

// GetUnreadCount returns the number of unread notifications.
func (c *Client) GetUnreadCount(ctx context.Context) (int, error) {
	var out bsky.NotificationGetUnreadCount_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.notification.getUnreadCount", map[string]any{}, &out)
	})
	if err != nil {
		return 0, wrapErr("GetUnreadCount", err)
	}
	return int(out.Count), nil
}

// MarkNotificationsRead marks all notifications as read up to the given time.
func (c *Client) MarkNotificationsRead(ctx context.Context, seenAt time.Time) error {
	input := &bsky.NotificationUpdateSeen_Input{
		SeenAt: seenAt.UTC().Format(time.RFC3339),
	}
	err := withRetry(ctx, 2, func() error {
		return c.api.Post(ctx, "app.bsky.notification.updateSeen", input, nil)
	})
	if err != nil {
		return wrapErr("MarkNotificationsRead", err)
	}
	return nil
}
