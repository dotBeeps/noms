# noms

A terminal UI client for Bluesky with Voresky RP overlay support. Built in Go with Bubble Tea v2, noms brings the full Bluesky experience to the terminal — timeline browsing, posting, notifications, search, profiles — plus first-class Voresky integration for RP character enrichment.

## Goals

- Full-featured Bluesky TUI: feed, threads, compose, notifications, search, profiles
- First-class Voresky RP overlay: character management, enriched display names/avatars, RP notifications
- Rich terminal rendering: Kitty graphics protocol, Sixel, halfblock fallback for images
- Offline-resilient auth: full atproto OAuth with DPoP, PKCE, encrypted token persistence, auto-refresh
- Polished UX: themeable (11+ built-in themes), keyboard-driven, responsive layout
- DM/chat support via atproto's chat API
- Prefer Charm ecosystem libraries (Bubbles, Huh, etc.) over hand-rolled components when suitable abstractions exist

## Non-Goals

- Not a general-purpose atproto client — Bluesky-specific features are the priority
- Not a mobile or web app — terminal-only
- No bot or automation features — this is for interactive human use
- No plugin/extension system — features are built-in

## Architecture

### Entry Point

`cmd/noms/main.go` creates `ui.App` and runs it via Bubble Tea.

### App Shell

`internal/ui/app.go` is the central model managing screen navigation via a `Screen` enum (Login, Feed, Notifications, Profile, Search, Compose, Thread, VoreskySetup, Voresky, VoreskyNotifications). Owns the Bluesky client, Voresky client, auth session, image cache, and enrichment manager.

### API Layer

- `internal/api/bluesky/` — Bluesky/atproto API client (interface-based for testability), wraps indigo's `atclient.APIClient`, includes retry logic
- `internal/api/voresky/` — Voresky RP overlay API, cookie-based auth with auto-revalidation

### Auth

`internal/auth/` — Full atproto OAuth: DPoP proofs, PKCE, loopback HTTP redirect server, encrypted session persistence, token refresh

### UI Layer

Each screen is its own package under `internal/ui/`:

| Package            | Purpose                                           |
| ------------------ | ------------------------------------------------- |
| `feed/`            | Timeline with post rendering, optimistic actions  |
| `thread/`          | Thread/reply view                                 |
| `notifications/`   | Bluesky notifications                             |
| `vnotifications/`  | Voresky RP notifications                          |
| `profile/`         | User profile view                                 |
| `search/`          | Post and actor search                             |
| `compose/`         | Post composer with reply support                  |
| `login/`           | OAuth login flow                                  |
| `vsetup/`, `vtab/` | Voresky setup and tab views                       |
| `components/`      | Shared UI (tab bar, status bar, help overlay)     |
| `shared/`          | Rendering utilities, scroll helpers               |
| `theme/`           | Color palette system                              |
| `images/`          | Kitty/Sixel/halfblock image rendering + LRU cache |
| `enrichment/`      | Voresky display name/avatar overrides             |

### Config

`internal/config/` — TOML config at `~/.config/noms/config.toml` (theme, default account, image protocol). Keyring integration for credentials.

## Tech Stack

| Layer     | Choice              | Why                                                |
| --------- | ------------------- | -------------------------------------------------- |
| Language  | Go 1.26             | Fast compilation, single binary, great concurrency |
| Framework | Bubble Tea v2       | Best-in-class TUI framework, MVU architecture      |
| Styling   | Lipgloss v2         | Composable terminal styling from same ecosystem    |
| atproto   | indigo              | Go atproto SDK with full API coverage              |
| Images    | go-termimg          | Multi-protocol terminal image rendering            |
| Config    | BurntSushi/toml     | Simple, human-readable config format               |
| Auth      | golang-jwt + crypto | DPoP proof generation for atproto OAuth            |
| Build     | go build            | No external build tools needed                     |
| Test      | go test             | Stdlib testing with interface-based mocking        |

## Key Decisions

1. **Interface-based API mocking** — `BlueskyClient` is an interface to enable fast, networkless UI testing
2. **Optimistic updates** — Like/repost/delete apply UI changes immediately, roll back on API failure
3. **Message-based architecture** — Standard Bubble Tea Msg routing through the App shell
4. **No CI pipeline** — Manual build and test; goreleaser for releases
5. **Kitty graphics priority** — Kitty protocol is the primary image path, with Sixel and halfblock as fallbacks
6. **Voresky as first-class overlay** — Not a plugin; deeply integrated into feed rendering and navigation
7. **Huh for structured input flows** — Theme picker, login chooser, vsetup cookie entry, and login form use `huh` forms/selects instead of hand-rolled cursor+key state machines
8. **Harmonica for all animation** — Spring physics via `harmonica` replaces hand-rolled `Decay`/ease-out math for toast fade, badge bounce, tab transitions, scroll easing, and action feedback
9. **Styles as factory functions** — Lipgloss styles that reference `theme.Color*` vars use zero-arg factory functions (not package-level vars) so they pick up the current theme after `Apply()`. Only styles with no theme dependency may be package-level vars
10. **AdaptiveColor for light terminals** — Theme palette entries use `lipgloss.AdaptiveColor` with dark/light variants instead of dark-only values
11. **No skeleton loading** — Rejected; spinners and empty states are preferred over skeleton placeholders
