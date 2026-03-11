# Smoke Test Protocol

Manual testing checklist for noms TUI against a real Bluesky account.

## Prerequisites

- Go 1.25+
- Terminal with 80x24 minimum
- Bluesky account (test account recommended)
- Working internet connection

## Build

```bash
go build -o ./noms ./cmd/noms/
```

## Version Check

```bash
./noms --version
# Expected: noms v0.1.0-dev
```

## Test Checklist

### 1. Login

- [ ] Launch `./noms`
- [ ] Login screen shows Handle/Password fields
- [ ] Press `?` to toggle help overlay
- [ ] Enter credentials and authenticate
- [ ] After login: switches to Feed screen, tab bar shows `[1] Feed` active
- [ ] Status bar shows handle and "Connected"

### 2. Feed Screen

- [ ] Timeline posts load and render with author, text, engagement counts
- [ ] `j`/`k` or arrow keys navigate post selection
- [ ] Scrolling loads more posts near bottom (pagination)
- [ ] `l` toggles like on selected post (count updates)
- [ ] `t` toggles repost on selected post (count updates)
- [ ] `Enter` on a post opens Thread view
- [ ] `c` opens Compose (new post)
- [ ] `r` on a post opens Compose (reply)

### 3. Thread Screen

- [ ] Thread loads with parent chain and replies
- [ ] Target post has highlighted border
- [ ] `j`/`k` navigate between posts in thread
- [ ] `l` likes selected post
- [ ] `t` reposts selected post
- [ ] `r` opens reply compose for selected post
- [ ] `p` opens profile of selected post's author
- [ ] `Esc` or `Backspace` returns to previous screen
- [ ] `Enter` on a reply opens that reply's thread

### 4. Compose Screen

- [ ] New post: header shows "New Post"
- [ ] Reply: header shows "Reply", parent post displayed above
- [ ] Textarea accepts text input
- [ ] Character counter updates (shows X/300)
- [ ] `Ctrl+Enter` submits post
- [ ] After submit: returns to previous screen, feed refreshes
- [ ] `Esc` cancels and returns to previous screen
- [ ] Empty post shows error on submit

### 5. Notifications Screen (Tab 2)

- [ ] Press `2` to switch to Notifications
- [ ] Notifications load (likes, reposts, follows, replies, mentions)
- [ ] Unread count shows in header and status bar badge
- [ ] `j`/`k` navigate notification list
- [ ] `Enter` on like/repost/reply/mention navigates to post thread
- [ ] `Enter` on follow navigates to profile
- [ ] `r` refreshes and marks as read

### 6. Profile Screen (Tab 3)

- [ ] Press `3` to view own profile
- [ ] Display name, handle, bio, follower/following/post counts render
- [ ] Author's posts load below profile header
- [ ] `j`/`k` navigate posts
- [ ] `Enter` on a post opens thread
- [ ] `r` refreshes profile and feed
- [ ] Navigate to another user's profile (from notification or thread)
- [ ] `f` toggles follow on other users' profiles
- [ ] `Esc` returns to previous screen

### 7. Search Screen (Tab 4)

- [ ] Press `4` to switch to Search
- [ ] Text input is focused, type to search
- [ ] Results appear after typing (debounced)
- [ ] `Tab` toggles between Posts and People modes
- [ ] Post results: `Enter` opens thread
- [ ] People results: `Enter` opens profile
- [ ] `Esc` blurs input, second `Esc` clears search
- [ ] `/` refocuses input

### 8. Navigation & Chrome

- [ ] Tab switching with `1`-`4` works from any main screen
- [ ] Tab bar highlights active tab
- [ ] `?` toggles help overlay from any screen (except compose)
- [ ] `Ctrl+C` exits from any screen
- [ ] Window resize updates layout
- [ ] Status bar persists across all screens

## Known Limitations

- OAuth browser flow requires redirect URI setup
- Rate limiting may cause temporary API errors (auto-retried)
- Image/embed content not rendered (text-only)
- No offline caching
- Search requires typing at least a few characters before results appear
