# Image Rendering

`internal/ui/images/cache.go` and `renderer.go`

---

## Protocol Detection

On startup, `images.New()` calls `termimg.DetectProtocol()` from [go-termimg](https://github.com/blacktop/go-termimg) to probe the terminal. Detection order: **Kitty → Sixel → Halfblocks → Unsupported**.

The detected protocol is stored in `Cache.protocol`. `Cache.Enabled()` returns false only if the protocol is `Unsupported`.

For Kitty, noms opens `/dev/tty` directly for writing APC sequences. If that fails, it falls back to Halfblocks.

Override with `image_protocol` in config.

---

## Lazy Rendering

Images are fetched asynchronously. When a post with an avatar or embed image is first rendered, noms issues a `Fetch` command (a `tea.Cmd`). The command runs in a background goroutine and emits `ImageFetchedMsg{URL}` when done.

On receiving `ImageFetchedMsg`, the app triggers a re-render of affected items. Until the image is fetched, a blank placeholder or text fallback is shown.

---

## Kitty Graphics Protocol

For Kitty terminals, noms bypasses Bubble Tea's rendering pipeline and writes directly to `/dev/tty`:

1. **Transmit**: PNG data is base64-encoded and sent in 4096-byte chunks as Kitty APC sequences (`\x1b_G...m=1;...\x1b\` / `\x1b_Gm=0;...\x1b\`). Each image gets a unique numeric ID from a global atomic counter.

2. **Virtual placement**: After transmitting, noms sends a virtual placement command (`\x1b_Ga=p,U=1,i={id},c={cols},r={rows},q=2\x1b\`) to size the image.

3. **Placeholders**: The rendered string contains Unicode placeholder characters (`U+10EEEE`) that Kitty replaces with the transmitted image.

Placement is always re-sent on render to handle terminal-side cache eviction (Kitty can evict images from its VRAM if it runs out of space).

---

## Generation-Based Invalidation

`Cache.generation` is a `uint32` incremented by `InvalidateTransmissions()`. This is called whenever the terminal size changes or the app reinitializes.

When rendering a Kitty image, noms checks the stored `transmittedEntry.generation` against the current generation. If stale, the image is re-transmitted with a fresh ID before placing.

Downloaded PNG files on disk are never invalidated — only the Kitty transmission IDs are reset.

---

## Retry & Backoff

Failed fetches are tracked in `Cache.failedFetches` with a cooldown:

| Attempt      | Retry delay |
| ------------ | ----------- |
| 1st retry    | 2 s         |
| 2nd retry    | 4 s         |
| 3rd retry    | 8 s         |
| Max delay    | 2 min       |
| Max cooldown | 5 min       |

4xx (client error) responses are not retried. After `maxFetchRetries` (3) exhausted, the URL enters cooldown and won't be retried until the cooldown expires.

`ClearFailedFetches()` resets all cooldowns immediately.

---

## Avatar Transform

`FetchAvatar()` applies a rounded-corner mask (30% radius) to downloaded images before saving. The corner fill color is the theme's `Surface` color, converted from ANSI to RGB via `theme.ANSIToRGB()` (`internal/ui/theme/ansi_rgb.go`).

---

## Image Dimensions

All images are resized to fit within **256×256 pixels** using Lanczos3 resampling (`github.com/nfnt/resize`). The aspect ratio is preserved. Images already within bounds are not resized.

Resized images are saved as PNG to a temp directory (`$TMPDIR/noms-images/`), keyed by a SHA-256 hash of the source URL.
