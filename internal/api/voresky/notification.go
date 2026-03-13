package voresky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NotificationType enumerates the known Voresky notification types.
type NotificationType string

const (
	NotifPoke                   NotificationType = "poke"
	NotifStalk                  NotificationType = "stalk"
	NotifStalkTargetAvailable   NotificationType = "stalk_target_available"
	NotifHousingInvite          NotificationType = "housing_invite"
	NotifHousingRequest         NotificationType = "housing_request"
	NotifHousingJoin            NotificationType = "housing_join"
	NotifHousingLeave           NotificationType = "housing_leave"
	NotifHousingKick            NotificationType = "housing_kick"
	NotifHousingInviteAccepted  NotificationType = "housing_invite_accepted"
	NotifHousingInviteRejected  NotificationType = "housing_invite_rejected"
	NotifHousingRequestAccepted NotificationType = "housing_request_accepted"
	NotifHousingRequestRejected NotificationType = "housing_request_rejected"
	NotifInteractionProposal    NotificationType = "interaction_proposal"
	NotifInteractionAccepted    NotificationType = "interaction_accepted"
	NotifInteractionRejected    NotificationType = "interaction_rejected"
	NotifInteractionCounter     NotificationType = "interaction_counter_proposal"
	NotifInteractionVipCaught   NotificationType = "interaction_vip_caught"
	NotifInteractionNodeChanged NotificationType = "interaction_node_changed"
	NotifInteractionEscaped     NotificationType = "interaction_escaped"
	NotifInteractionReleased    NotificationType = "interaction_released"
	NotifInteractionRetreated   NotificationType = "interaction_prey_retreated"
	NotifInteractionRespawning  NotificationType = "interaction_respawning"
	NotifInteractionCompleted   NotificationType = "interaction_completed"
	NotifInteractionSafeword    NotificationType = "interaction_safeword"
	NotifCollarOffer            NotificationType = "collar_offer"
	NotifCollarRequest          NotificationType = "collar_request"
	NotifCollarAccepted         NotificationType = "collar_accepted"
	NotifCollarRejected         NotificationType = "collar_rejected"
	NotifCollarBroken           NotificationType = "collar_broken"
	NotifCollarLockRequest      NotificationType = "collar_lock_request"
	NotifCollarLocked           NotificationType = "collar_locked"
	NotifCollarUnlocked         NotificationType = "collar_unlocked"
)

// NotificationCategory groups notification types for filtering.
type NotificationCategory string

const (
	CategoryPokes        NotificationCategory = "Pokes"
	CategoryStalking     NotificationCategory = "Stalking"
	CategoryHousing      NotificationCategory = "Housing"
	CategoryInteractions NotificationCategory = "Interactions"
	CategoryCollars      NotificationCategory = "Collars"
)

// NotificationCharacterInfo holds the character info attached to a notification.
type NotificationCharacterInfo struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Avatar  CharacterAvatar `json:"avatar"`
	UserDID string          `json:"userDid"`
}

// Notification represents a single Voresky notification with details.
type Notification struct {
	ID                       string                     `json:"id"`
	RecipientDID             string                     `json:"recipientDid"`
	Type                     NotificationType           `json:"type"`
	Payload                  json.RawMessage            `json:"payload"`
	Universe                 string                     `json:"universe"`
	IsRead                   bool                       `json:"isRead"`
	CreatedAt                time.Time                  `json:"createdAt"`
	ReadAt                   *time.Time                 `json:"readAt"`
	SourceCharacter          *NotificationCharacterInfo `json:"sourceCharacter"`
	TargetCharacter          *NotificationCharacterInfo `json:"targetCharacter"`
	RecipientMainCharacterID string                     `json:"recipientMainCharacterId"`
}

type notificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    string         `json:"nextCursor"`
}

type unreadCountResponse struct {
	UnreadCount int `json:"unreadCount"`
}

// GetNotifications fetches a page of Voresky notifications.
// Calls GET /api/game/notifications with optional limit and cursor.
func (c *VoreskyClient) GetNotifications(ctx context.Context, limit int, cursor string) ([]Notification, string, error) {
	var params []string
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if cursor != "" {
		params = append(params, "cursor="+cursor)
	}

	path := "/api/game/notifications"
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, "", fmt.Errorf("get notifications: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", ParseError(resp)
	}

	var nr notificationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		return nil, "", fmt.Errorf("decode notifications: %w", err)
	}
	return nr.Notifications, nr.NextCursor, nil
}

// GetUnreadNotificationCount returns the number of unread Voresky notifications.
// Calls GET /api/game/notifications/count.
func (c *VoreskyClient) GetUnreadNotificationCount(ctx context.Context) (int, error) {
	resp, err := c.Get(ctx, "/api/game/notifications/count")
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, ParseError(resp)
	}

	var ucr unreadCountResponse
	if err := json.NewDecoder(resp.Body).Decode(&ucr); err != nil {
		return 0, fmt.Errorf("decode unread count: %w", err)
	}
	return ucr.UnreadCount, nil
}

// MarkNotificationsRead marks the given notification IDs as read.
// Calls POST /api/game/notifications/read.
func (c *VoreskyClient) MarkNotificationsRead(ctx context.Context, ids []string) error {
	body := map[string]interface{}{"ids": ids}
	resp, err := c.Post(ctx, "/api/game/notifications/read", body)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return ParseError(resp)
	}
	return nil
}
