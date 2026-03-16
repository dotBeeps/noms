# Screens

## Feed (`ScreenFeed`, tab `1`)

The main timeline. Fetches the authenticated user's Bluesky home feed on load and on refresh.

**Key interactions:**

| Key       | Action                            |
| --------- | --------------------------------- |
| `j` / `k` | Scroll up/down                    |
| `Enter`   | Open thread                       |
| `l`       | Like / unlike (optimistic)        |
| `t`       | Repost / un-repost (optimistic)   |
| `r`       | Reply (opens Compose)             |
| `c`       | Compose new post                  |
| `d`       | Delete your own post (optimistic) |

Posts render author avatar (if image protocol enabled), display name, handle, timestamp, post text with rich text (links, mentions, tags), and embed previews. Voresky enrichment overlays character names and avatars when active.

---

## Thread (`ScreenThread`)

Shows a single post and its reply tree. Entered from Feed, Notifications, or Profile by pressing Enter on a post.

**Key interactions:**

| Key             | Action                  |
| --------------- | ----------------------- |
| `j` / `k`       | Scroll                  |
| `Enter`         | Open nested thread      |
| `l` / `t` / `r` | Like / repost / reply   |
| `p`             | View author's profile   |
| `d`             | Delete your post        |
| `Esc`           | Back to previous screen |

---

## Notifications (`ScreenNotifications`, tab `2`)

Lists Bluesky notifications: likes, reposts, mentions, replies, follows, and quotes.

**Key interactions:**

| Key       | Action                       |
| --------- | ---------------------------- |
| `j` / `k` | Scroll                       |
| `Enter`   | Open related post/thread     |
| `r`       | Refresh and mark all as read |

---

## Profile (`ScreenProfile`, tab `3`)

Shows a user's profile header (avatar, display name, handle, bio, follower counts) and their post history.

**Key interactions:**

| Key       | Action               |
| --------- | -------------------- |
| `j` / `k` | Scroll               |
| `Enter`   | Open thread          |
| `f`       | Follow / unfollow    |
| `r`       | Refresh              |
| `d`       | Delete your own post |
| `Esc`     | Back                 |

Navigated to from Thread (`p`), or by selecting a user in Search.

---

## Search (`ScreenSearch`, tab `4`)

Two modes: post search and actor (people) search. The active mode is shown in the tab bar area.

**Key interactions:**

| Key       | Action                           |
| --------- | -------------------------------- |
| `/`       | Focus the search input           |
| `Tab`     | Toggle between posts and people  |
| `j` / `k` | Scroll results                   |
| `Enter`   | Open post/thread or view profile |

---

## Compose (`ScreenCompose`)

Full-screen text editor for writing posts. Can be a new post or a reply (reply context shown at top).

**Key interactions:**

| Key          | Action            |
| ------------ | ----------------- |
| `Ctrl+Enter` | Submit post       |
| `Esc`        | Cancel and return |

After a successful post, noms returns to the previous screen and refreshes the feed.

---

## Login (`ScreenLogin`)

The initial screen. Prompts for a Bluesky handle. Pressing Enter starts the OAuth flow — opens the browser and waits for the loopback callback. See [`docs/auth.md`](auth.md) for the full flow.

---

## Voresky Setup (`ScreenVoreskySetup`)

Prompts for a Voresky session cookie. On success, the Voresky client is initialized, tabs 5 and 6 appear, and the app fetches the main character. See [`docs/voresky.md`](voresky.md).

---

## Voresky Tab (`ScreenVoresky`, tab `5`)

Lists Voresky characters for the authenticated user.

**Key interactions:**

| Key       | Action                 |
| --------- | ---------------------- |
| `j` / `k` | Scroll                 |
| `Enter`   | View character details |

Only visible after Voresky is configured.

---

## Voresky Notifications (`ScreenVoreskyNotifications`, tab `6`)

Lists RP-specific notifications from Voresky: interactions, housing events, collar events, pokes, and stalk notifications.

**Key interactions:**

| Key       | Action                |
| --------- | --------------------- |
| `j` / `k` | Scroll                |
| `r`       | Mark selected as read |

Only visible after Voresky is configured.
