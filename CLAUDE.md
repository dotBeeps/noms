# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**noms** is a TUI (terminal UI) client for Bluesky (atproto) with Voresky RP overlay support, written in Go (`github.com/dotBeeps/noms`, Go 1.26). It uses the Bubble Tea framework (v2) with lipgloss for styling.

## Build & Run

```bash
go build -o noms ./cmd/noms    # build binary
./noms                          # run
go test ./...                   # run all tests
go test ./internal/ui/feed      # run tests for a single package
go test ./internal/ui/feed -run TestRenderPost  # run a single test
```

There is no Makefile or CI pipeline. The binary is `noms` at the project root (gitignored). Version is set at build time via `-ldflags "-X main.version=..."` (defaults to `0.1.0-dev`).

## Architecture

### Entry Point & App Shell

`cmd/noms/main.go` — minimal main that creates `ui.App` and runs it via Bubble Tea.

`internal/ui/app.go` — central App model implementing `tea.Model`. Manages screen navigation via a `Screen` enum (Login, Feed, Notifications, Profile, Search, Compose, Thread, VoreskySetup, Voresky, VoreskyNotifications). Owns the Bluesky client, Voresky client, auth session, image cache, and enrichment manager. Routes messages to the active screen's sub-model.

**View lifecycle**: Some views are persistent (`feedModel`, `notifModel`, `searchModel`) and survive screen switches. Others (`profileModel`, `threadModel`, `composeModel`, `vsetupModel`) are created on demand. Lazy initialization flags (`notifInitialized`, `voreskyTabInit`, etc.) gate first-load data fetches.

### API Clients

- `internal/api/bluesky/` — Bluesky/atproto API. `BlueskyClient` is an **interface** (in `client.go`) enabling mock-based testing in UI packages. Wraps `indigo`'s `atclient.APIClient`. Includes retry logic for rate limits.
- `internal/api/voresky/` — Voresky RP overlay API. Cookie-based auth with auto-revalidation on 401. Character management, enrichment (RP display names/avatars), and notification payloads.

### Auth

`internal/auth/` — Full atproto OAuth flow: DPoP proof generation, PKCE, loopback HTTP server for redirect, session persistence (encrypted tokens on disk), and token refresh.

### UI Layer (Bubble Tea)

All UI lives under `internal/ui/`. Each screen is its own package with a model implementing `tea.Model`:

- `feed/` — Timeline view with post rendering (`post_widget.go`), optimistic like/repost/delete actions (`actions.go`)
- `thread/` — Thread/reply view
- `notifications/`, `vnotifications/` — Bluesky and Voresky notification views
- `profile/` — User profile view
- `search/` — Post and actor search
- `compose/` — Post composer with reply support
- `login/` — OAuth login flow
- `vsetup/`, `vtab/` — Voresky setup and tab views
- `components/` — Shared UI components (tab bar, status bar, help overlay)
- `shared/` — Shared utilities: `RenderItemWithBorder` (bordered panel rendering with ANSI-safe background), `EnsureSelectedVisible` (scroll offset calculation), Kitty placeholder detection, text joining helpers
- `theme/` — Color palette system with named themes (default, dracula, nord, tokyo-night, rose-pine, etc.). Uses `Apply()` to set package-level `Color*` and `Style*` vars.
- `images/` — Image rendering with Kitty graphics protocol support and an LRU cache
- `enrichment/` — Voresky enrichment manager (RP display name/avatar overrides)

### Config

`internal/config/` — TOML config at `~/.config/noms/config.toml`. Handles theme selection, default account, image protocol. Keyring integration for credential storage.

## Key Patterns

- **Interface-based API mocking**: `bluesky.BlueskyClient` interface allows all UI tests to run without network calls. Tests create mock structs implementing this interface.
- **Optimistic updates**: Like/repost/delete apply UI changes immediately, then issue API calls. On failure, changes are rolled back.
- **Message-based architecture**: Bubble Tea Msg types are defined at the top of each screen's file. The App routes messages to the active screen model.
- **ANSI background stabilization**: `RenderItemWithBorder` in `shared/scroll.go` manually re-injects background color sequences after ANSI resets to prevent lipgloss width miscalculations with Kitty placeholder characters.

## Testing Patterns

- All tests use `t.Parallel()`.
- Mock helpers are local to each test file (e.g., `createTestPost()`, `stripAnsi()`, `strPtr()` in feed tests; `newTestServer()`/`newTestClient()` in API tests).
- `cmd/noms/integration_test.go` has a full `newMockClient()` implementing the `BlueskyClient` interface — useful as a reference when writing new mocks.
- `stubImageRenderer` is used in feed/notification tests to stub out Kitty image rendering.
- Manual QA checklist: `docs/smoke-test.md`.
