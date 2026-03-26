package bluesky

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

// BlueskyClient defines the interface for all Bluesky/atproto API operations.
// This interface enables mocking in UI layer tests.
type BlueskyClient interface {
	// Feed
	GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error)
	GetPost(ctx context.Context, uri string) (*bsky.FeedDefs_PostView, error)
	GetPostThread(ctx context.Context, uri string, depth int) (*bsky.FeedGetPostThread_Output, error)

	// Profile
	GetProfile(ctx context.Context, actor string) (*bsky.ActorDefs_ProfileViewDetailed, error)
	GetAuthorFeed(ctx context.Context, actor string, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error)
	FollowActor(ctx context.Context, did string) error
	UnfollowActor(ctx context.Context, did string) error

	// Notifications
	ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error)
	GetUnreadCount(ctx context.Context) (int, error)
	MarkNotificationsRead(ctx context.Context, seenAt time.Time) error

	// Post actions
	CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error)
	DeletePost(ctx context.Context, uri string) error
	Like(ctx context.Context, uri, cid string) (string, error)
	Unlike(ctx context.Context, likeURI string) error
	Repost(ctx context.Context, uri, cid string) (string, error)
	UnRepost(ctx context.Context, repostURI string) error

	// Search
	SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error)
	SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error)
}

// Client is the concrete Bluesky API client backed by indigo's APIClient.
type Client struct {
	api *atclient.APIClient
	did string
}

// NewClient creates a new Bluesky API client.
//
// httpClient should be an authenticated *http.Client (e.g. from Session.AuthenticatedHTTPClient()).
// pdsURL is the user's PDS host (e.g. "https://bsky.social").
// did is the authenticated user's DID.
func NewClient(httpClient *http.Client, pdsURL, did string) *Client {
	api := atclient.NewAPIClient(pdsURL)
	api.Client = httpClient
	return &Client{api: api, did: did}
}

// NewClientFromAPI creates a new Bluesky API client from a pre-authenticated atclient.APIClient.
// Use this when authenticating via app password (atclient.LoginWithPassword returns *APIClient directly).
func NewClientFromAPI(api *atclient.APIClient, did string) *Client {
	return &Client{api: api, did: did}
}

// DID returns the authenticated user's DID.
func (c *Client) DID() string {
	return c.did
}

// APIClient returns the underlying atclient.APIClient for advanced usage.
func (c *Client) APIClient() *atclient.APIClient {
	return c.api
}

// RateLimit holds parsed rate limit information from API response headers.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// parseRateLimit extracts rate limit info from HTTP response headers.
func parseRateLimit(h http.Header) *RateLimit {
	limitStr := h.Get("RateLimit-Limit")
	remainStr := h.Get("RateLimit-Remaining")
	resetStr := h.Get("RateLimit-Reset")

	if limitStr == "" && remainStr == "" && resetStr == "" {
		return nil
	}

	rl := &RateLimit{}
	if v, err := strconv.Atoi(limitStr); err == nil {
		rl.Limit = v
	}
	if v, err := strconv.Atoi(remainStr); err == nil {
		rl.Remaining = v
	}
	if v, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
		rl.Reset = time.Unix(v, 0)
	}
	return rl
}

// isRateLimited checks if an error is a 429 rate limit error.
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := asAPIError(err); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

// asAPIError attempts to extract an *atclient.APIError from an error.
func asAPIError(err error) (*atclient.APIError, bool) {
	var apiErr *atclient.APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// withRetry executes fn with exponential backoff on rate limit (429) errors.
// It retries up to maxRetries times with exponential backoff.
func withRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	backoff := 500 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRateLimited(lastErr) {
			return lastErr
		}
		if attempt == maxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return lastErr
}

// wrapErr wraps an error with method context.
func wrapErr(method string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", method, err)
}

// parseATURI splits an AT-URI (at://did/collection/rkey) into its components.
func parseATURI(uri string) (repo, collection, rkey string, err error) {
	if !strings.HasPrefix(uri, "at://") {
		return "", "", "", fmt.Errorf("invalid AT-URI: must start with at://: %s", uri)
	}
	parts := strings.SplitN(uri[5:], "/", 3)
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid AT-URI: expected at://repo/collection/rkey: %s", uri)
	}
	return parts[0], parts[1], parts[2], nil
}

// IsRateLimited reports whether err is a 429 rate limit error.
func IsRateLimited(err error) bool {
	return isRateLimited(err)
}

// IsNetworkError reports whether err is a network-level failure
// (timeout, DNS, connection refused, etc.) as opposed to an API error.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// If it's a known API error (got an HTTP response), it's not a network error
	if _, ok := asAPIError(err); ok {
		return false
	}
	// Context deadline/cancellation counts as network-level
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	// Check for net.Error (timeouts, DNS failures, connection refused)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// Unwrap and check for url.Error (wraps net errors from http.Client)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	return false
}

// Compile-time interface assertion.
var _ BlueskyClient = (*Client)(nil)
