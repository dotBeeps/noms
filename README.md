# noms

A terminal UI client for [Bluesky](https://bsky.app) with [Voresky](https://voresky.app) RP overlay support, written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

## Features

- **Feed** — scrollable timeline with post rendering, images, rich text
- **Threads** — full thread/reply view
- **Notifications** — Bluesky notifications (likes, replies, reposts, mentions, follows)
- **Search** — post and actor search
- **Compose** — write posts and replies
- **Profiles** — view user profiles and their posts
- **Voresky tab** — RP character management and enriched feed display
- **Voresky notifications** — RP-specific notifications (interactions, housing, collars, pokes, stalking)
- **Image rendering** — Kitty graphics protocol, Sixel, and halfblock fallback
- **Themes** — 11 built-in color themes

---

## Install & Build

### Pre-built releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/dotBeeps/noms/releases).

Packages are named `noms_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows). Extract and put the `noms` binary somewhere on your `$PATH`.

Available targets: `linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64`, `windows_amd64`.

### Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/dotBeeps/noms
cd noms
go build -o noms ./cmd/noms
./noms
```

---

## First Run & Login

On first launch, noms shows the login screen. Enter your Bluesky handle (e.g. `you.bsky.social`) and press Enter. A browser window opens for the standard atproto OAuth flow. After authorizing, noms receives the callback automatically via a local loopback server.

Session tokens are stored encrypted on disk at `~/.local/share/noms/tokens.enc`.

---

## Key Bindings

### Tab switching

| Key | Tab                        |
| --- | -------------------------- |
| `1` | Feed                       |
| `2` | Notifications              |
| `3` | Profile                    |
| `4` | Search                     |
| `5` | Voresky _(if configured)_  |
| `6` | V-Notifs _(if configured)_ |

### Navigation (most screens)

| Key       | Action        |
| --------- | ------------- |
| `j` / `↓` | Down          |
| `k` / `↑` | Up            |
| `Enter`   | Open / select |
| `Esc`     | Back          |

### Feed & Thread

| Key | Action                              |
| --- | ----------------------------------- |
| `l` | Like / unlike                       |
| `t` | Repost / un-repost                  |
| `r` | Reply                               |
| `c` | Compose new post _(feed only)_      |
| `p` | View author profile _(thread only)_ |
| `d` | Delete your post                    |

### Other screens

| Key          | Action                                |
| ------------ | ------------------------------------- |
| `r`          | Refresh / mark read _(notifications)_ |
| `f`          | Follow / unfollow _(profile)_         |
| `/`          | Focus search input                    |
| `Tab`        | Toggle posts ↔ people _(search)_     |
| `Ctrl+Enter` | Submit post _(compose)_               |

---

## Configuration

Config file: `~/.config/noms/config.toml` (created on first run with defaults).

```toml
default_account = ""       # DID or handle to auto-login on startup

[theme]
name = "default"           # theme name — see Themes below

image_protocol = "auto"    # auto | kitty | sixel | none
```

See [`docs/config.md`](docs/config.md) for full details and environment variables.

---

## Themes

Set `[theme] name` in config. Available:

`default` · `terminal` · `dracula` · `nord` · `tokyo-night` · `rose-pine` · `forest-night` · `neon-ember` · `retro-amber` · `iceberg` · `mint-latte`

Many aliases are accepted (`tokyonight`, `rosepine`, `ansi`, etc.). See [`docs/themes.md`](docs/themes.md).

---

## Voresky Setup

Voresky is an RP overlay for Bluesky. Enter your Voresky session cookie via the setup screen to enable tabs 5 (Voresky) and 6 (V-Notifs). With Voresky active, character display names and avatars are overlaid on the feed via the enrichment system.

See [`docs/voresky.md`](docs/voresky.md).

---

## Contributing & Testing

```bash
go build ./...                                       # verify it compiles
go test ./...                                        # run all tests
go test ./internal/ui/feed                           # single package
go test ./internal/ui/feed -run TestRenderPost       # single test
```

See [`docs/testing.md`](docs/testing.md) and [`docs/architecture.md`](docs/architecture.md).
