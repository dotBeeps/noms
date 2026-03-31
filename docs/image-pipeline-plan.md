# Image Pipeline Overhaul Plan

Hardening the image rendering pipeline for correctness, performance, and resource management.

Green nodes in `docs/goal-image-pipeline.svg` indicate new/changed components.

## Issues Found

| ID  | Severity | Issue                                         | Root Cause                                                                                       |
| --- | -------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| H1  | High     | No fetch concurrency limit                    | Each `tea.Cmd` spawns a goroutine; 50+ images per page load                                      |
| H2  | High     | TTY writes block Update path                  | `transmitImageData` writes hundreds of KB synchronously to `/dev/tty` during `rebuildViewport()` |
| H3  | High     | Rapid full viewport rebuilds                  | Each `ImageFetchedMsg` triggers `rebuildViewport()` in 4-6 models                                |
| H4  | High     | No cache eviction                             | All sync.Maps grow unbounded; ImageWidget holds decoded pixels                                   |
| H5  | High     | Failed images never re-fetched on scroll-back | Fetches only fire on `FeedLoadedMsg`, not on viewport visibility                                 |
| H6  | High     | Orphaned Kitty image IDs leak terminal memory | No `a=d` sent on generation bump                                                                 |
| M1  | Medium   | Full re-transmission on resize                | Pixel data re-sent when only placement needs updating                                            |
| M2  | Medium   | 429 misclassified as 4xx                      | `statusCode >= 400 && < 500` catches rate limits                                                 |
| M3  | Medium   | No jitter on retry backoff                    | Synchronized retry waves (thundering herd)                                                       |
| M4  | Medium   | Broadcast to inactive models                  | Off-screen persistent models rebuild on every fetch                                              |
| M5  | Medium   | `ClearFailedFetches` data race                | Replaces `sync.Map` struct non-atomically                                                        |
| M6  | Medium   | Latent tty write race                         | No mutex/serialization on `/dev/tty` writes                                                      |
| L1  | Low      | Unbounded `failedFetches` map                 | No TTL/cleanup                                                                                   |
| L2  | Low      | `Bounds().Max` vs `Dx()/Dy()`                 | Correct in practice but not defensive                                                            |
| L3  | Low      | `imageNumCounter` uint32 wrap                 | Theoretical overflow to invalid ID 0                                                             |

## Implementation Tiers

### Tier D: Quick Fixes (Low Risk)

Fixes: M5, M2, M3

| Fix                         | Change                                                | Files      |
| --------------------------- | ----------------------------------------------------- | ---------- |
| D1: ClearFailedFetches race | `Range` + `Delete` instead of struct replacement      | `cache.go` |
| D2: 429 handling            | Exempt 429 from 4xx break; honor `Retry-After` header | `cache.go` |
| D3: Retry jitter            | Full jitter via `math/rand/v2` in `retryDelay()`      | `cache.go` |

### Tier B: Behavioral Improvements (Medium Risk)

Fixes: H5, H1, H3, M4

| Fix                     | Change                                                                  | Files                        |
| ----------------------- | ----------------------------------------------------------------------- | ---------------------------- |
| B1: Viewport re-fetch   | `rebuildViewport()` returns `[]tea.Cmd` for visible-but-uncached images | `feed.go`, all screen models |
| B2: Fetch semaphore     | Buffered channel (cap=6) in Cache, acquired in `fetchWithTransform`     | `cache.go`                   |
| B3: Rebuild debouncing  | Dirty flag + 50ms `tea.Tick` coalescing in App                          | `app.go`                     |
| B4: Lazy model rebuilds | Per-model dirty flags; only rebuild active screen; flush on tab switch  | `app.go`, screen models      |

### Tier A: Architecture (High Impact)

Fixes: H2, H6, M1, M6, H4

| Fix                        | Change                                                                                                    | Files                                        |
| -------------------------- | --------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| A1: Async TTY transmission | Background goroutine owns all `/dev/tty` writes via channel queue                                         | `cache.go`, `renderer.go`, `app.go`          |
| A2: Smart invalidation     | Split `InvalidatePlacements()` (resize) vs `InvalidateTransmissions()` (theme). Proper `a=d` cleanup.     | `cache.go`, `renderer.go`, all screen models |
| A3: LRU eviction           | Replace 5 sync.Maps with mutex-guarded LRU (container/list, cap=200). Eviction cleans disk + sends `a=d`. | `cache.go`                                   |

## Dependency Graph

```
D1, D2, D3  (independent, ship first)
     |
B1 -> B2 -> B3 + B4
     |
A1 -> A2
     |
    A3
```

## Spec Tracking

After each tier, `docs/image-pipeline.svg` is updated to reflect the new current state.

| Tier | Status | Notes                                                                                                                                                                                                                           |
| ---- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| D    | Done   | ClearFailedFetches race fixed (Range+Delete), 429 handling with Retry-After, full jitter [base/2, base) on retries                                                                                                              |
| B    | Done   | Viewport re-fetch (rebuildViewport returns []tea.Cmd), semaphore (cap=6), 50ms debounce coalescing, per-model dirty flags with tab-switch flush                                                                                 |
| A    | Done   | A2: Kitty a=d cleanup on InvalidateTransmissions + stale generation re-transmit. A3: FIFO eviction (cap=200) with disk+Kitty+widget cleanup. A1 (async TTY) deferred — B-tier semaphore+debounce already bounds the perf issue. |
