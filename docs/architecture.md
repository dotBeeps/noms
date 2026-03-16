# Architecture

## App Shell

`cmd/noms/main.go` is minimal — creates a `ui.App` and hands it to `tea.NewProgram`.

`internal/ui/app.go` is the central model. It implements `tea.Model` and owns:

- The active `Screen` enum value
- Bluesky client (`bluesky.BlueskyClient` interface)
- Voresky client and auth
- Image cache (`*images.Cache`)
- Enrichment manager
- All persistent and on-demand sub-models

### Screen enum

```go
const (
    ScreenLogin Screen = iota
    ScreenFeed
    ScreenNotifications
    ScreenProfile
    ScreenSearch
    ScreenCompose
    ScreenThread
    ScreenVoreskySetup
    ScreenVoresky
    ScreenVoreskyNotifications
)
```

The tab bar always reflects the current screen. Tab keys `1`–`6` map directly to screen constants.

---

## Persistent vs On-Demand Views

**Persistent** — survive screen switches, re-used on every visit:

- `feedModel` (`feed.FeedModel`)
- `notifModel` (`notifications.NotificationsModel`)
- `searchModel` (`search.SearchModel`)
- `voreskyTabModel` (`vtab.VoreskyModel`)
- `vnotifModel` (`vnotifications.VNotificationsModel`)

**On-demand** — created fresh each time the screen is entered:

- `profileModel` (`profile.ProfileModel`)
- `threadModel` (`thread.ThreadModel`)
- `composeModel` (`compose.ComposeModel`)
- `vsetupModel` (`vsetup.VSetupModel`)

Lazy initialization flags (`notifInitialized`, `voreskyTabInit`, etc.) gate first-load data fetches so the network call only happens once.

---

## Message Routing

Bubble Tea delivers every `tea.Msg` to `App.Update`. The app routes content messages to the currently active screen's sub-model:

```
tea.Msg → App.Update
    → tabBar.Update (always)
    → statusBar.Update (always)
    → active screen model.Update (content messages only)
    → cross-screen messages handled directly (LikePostMsg, RepostMsg, etc.)
```

Screen-specific messages (e.g. `feed.OpenThreadMsg`, `thread.BackMsg`, `compose.PostCreatedMsg`) trigger screen transitions inside `App.Update`.

---

## Optimistic Updates + Rollback

Like, repost, and delete apply state changes immediately in the UI before the API call returns. The sequence:

1. User presses `l`/`t`/`d`
2. UI flips the local boolean (liked, reposted) or removes the item
3. A `tea.Cmd` fires the API call in the background
4. On error: a `FailedMsg` is dispatched, the state is reverted and an error is shown in the status bar

Both `feedModel` and `threadModel` handle this pattern identically via `actions.go`.

---

## Interface-Based API Mocking

`internal/api/bluesky/client.go` defines `BlueskyClient` as an interface. The real implementation (`*bluesky.Client`) wraps indigo's `atclient.APIClient`. All UI packages accept `BlueskyClient` rather than the concrete type.

This means every screen test creates a local mock struct implementing the interface — no network, no test server. The canonical example is `cmd/noms/integration_test.go`, which has a full `newMockClient()`.

The Voresky client (`*voresky.VoreskyClient`) is a concrete struct; its tests use `httptest.Server` instead.

---

## Key Message Flow (text diagram)

```
User input (key press)
    │
    ▼
tea.KeyMsg → App.Update
    │
    ├─ Tab key? → set screen, update tab bar
    │
    ├─ Content key? → forward to active screen model
    │                    │
    │                    └─ screen emits Cmd (API call)
    │                                │
    │                                ▼
    │                         background goroutine
    │                                │
    │                                ▼
    └─ Result Msg → App.Update → route to screen model → UI update
```

---

## Chrome: Tab Bar & Status Bar

`internal/ui/components/tabbar.go` — renders the `[1] Feed [2] Notifications ...` row. Voresky tabs only appear when `VoreskyActive` is true.

`internal/ui/components/statusbar.go` — one-line bar below the content. Shows the active account handle and transient status messages (errors, loading states).

Both are persistent sub-models updated on every `tea.Msg`.

---

## ANSI Background Stabilization

`internal/ui/shared/scroll.go` (`RenderItemWithBorder`) manually re-injects background color ANSI sequences after each reset. This is necessary because lipgloss's width calculation can be thrown off by Kitty image placeholder characters (`\U0010EEEE`), which contain embedded ANSI. Without re-injection, the surface background bleeds or disappears on selected items.
