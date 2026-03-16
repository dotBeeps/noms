# Configuration

## Config File

Location: `~/.config/noms/config.toml`

Created automatically with defaults on first run. Edit manually to change settings.

### Full schema

```toml
# Bluesky DID or handle to auto-login on startup.
# Leave empty to show the login screen every time.
default_account = ""

[theme]
# Theme name. See docs/themes.md for available themes.
name = "default"

# Terminal graphics protocol for image rendering.
# auto   — detect automatically (tries Kitty, then Sixel, then halfblocks)
# kitty  — force Kitty graphics protocol
# sixel  — force Sixel
# none   — disable image rendering
image_protocol = "auto"
```

---

## XDG Directory Layout

noms follows the XDG Base Directory spec. All directories are created automatically.

| Purpose                 | Default path           | Override env var  |
| ----------------------- | ---------------------- | ----------------- |
| Config                  | `~/.config/noms/`      | `XDG_CONFIG_HOME` |
| Data (tokens, DPoP key) | `~/.local/share/noms/` | `XDG_DATA_HOME`   |
| Cache                   | `~/.cache/noms/`       | `XDG_CACHE_HOME`  |

Files in the data directory:

| File         | Contents                               |
| ------------ | -------------------------------------- |
| `tokens.enc` | AES-256-GCM encrypted session tokens   |
| `dpop.pem`   | EC P-256 DPoP signing key (PEM format) |

---

## Environment Variables

| Variable          | Effect                                                                                                                                                                        |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `XDG_CONFIG_HOME` | Base directory for config (default: `~/.config`)                                                                                                                              |
| `XDG_DATA_HOME`   | Base directory for data (default: `~/.local/share`)                                                                                                                           |
| `XDG_CACHE_HOME`  | Base directory for cache (default: `~/.cache`)                                                                                                                                |
| `NOMS_TOKEN_KEY`  | Passphrase for token file encryption. Defaults to a SHA-256 hash derived from the user's home directory path (per-machine key). Set this for stronger or portable encryption. |
| `NOMS_DEBUG`      | Enable image subsystem debug logging to `/tmp/noms-images-debug.log`                                                                                                          |

---

## image_protocol values

| Value   | Behavior                                                                                  |
| ------- | ----------------------------------------------------------------------------------------- |
| `auto`  | Calls `termimg.DetectProtocol()` at startup. Tries Kitty → Sixel → Halfblocks → disabled. |
| `kitty` | Forces Kitty graphics protocol. Requires a Kitty-compatible terminal.                     |
| `sixel` | Forces Sixel.                                                                             |
| `none`  | Disables all image rendering.                                                             |

The detected protocol is shown in the status bar debug info (visible with `NOMS_DEBUG`).
