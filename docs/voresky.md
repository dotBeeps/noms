# Voresky

[Voresky](https://voresky.app) is an RP (roleplay) overlay service for Bluesky. noms integrates with its REST API for character management, enrichment, and notifications.

---

## Setup

1. Log into Voresky in a browser and copy your session cookie.
2. In noms, navigate to the Voresky setup screen and paste the cookie.
3. noms initializes the Voresky client and fetches your main character.
4. Tabs `5` (Voresky) and `6` (V-Notifs) appear in the tab bar.

The Voresky base URL is `https://voresky.app`.

---

## Cookie Auth

`internal/api/voresky/client.go` — `VoreskyClient` injects the session cookie on every request via the `Cookie` header.

On a `401 Unauthorized` response, the client calls `VoreskyAuth.RefreshOrRevalidate()` and retries the request once. This handles session expiry transparently.

`VoreskyAuth` manages the cookie string with a mutex for concurrent access.

---

## Enrichment System

`internal/api/voresky/enrich.go` and `internal/ui/enrichment/manager.go`

When Voresky is active, the enrichment manager overlays RP display names and avatars on posts and profiles. For each DID in the feed, the enrichment system can substitute:

- **Display name** — the character's RP name instead of the Bluesky display name
- **Avatar** — the character's avatar instead of the Bluesky avatar

The `EnrichmentManager` caches enrichment data and fetches missing entries in batches. Data is keyed by DID. The feed and profile screens call `manager.Enrich(did)` before rendering each post header.

---

## Character Management

`internal/api/voresky/character.go`

Characters are the core entity in Voresky. Each user can have multiple characters. The app fetches the "main character" on Voresky setup and displays it in the Voresky tab.

`Character` fields include: ID, name, avatar (`CharacterAvatar`), user DID, and RP metadata.

---

## Notifications

`internal/api/voresky/notification.go` and `notification_payloads.go`

Voresky notifications are grouped into five categories:

| Category         | Types                                                                                                                                                                                                                                                                                                                |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Pokes**        | `poke`                                                                                                                                                                                                                                                                                                               |
| **Stalking**     | `stalk`, `stalk_target_available`                                                                                                                                                                                                                                                                                    |
| **Housing**      | `housing_invite`, `housing_request`, `housing_join`, `housing_leave`, `housing_kick`, `housing_invite_accepted`, `housing_invite_rejected`, `housing_request_accepted`, `housing_request_rejected`                                                                                                                   |
| **Interactions** | `interaction_proposal`, `interaction_accepted`, `interaction_rejected`, `interaction_counter_proposal`, `interaction_vip_caught`, `interaction_node_changed`, `interaction_escaped`, `interaction_released`, `interaction_prey_retreated`, `interaction_respawning`, `interaction_completed`, `interaction_safeword` |
| **Collars**      | `collar_offer`, `collar_request`, `collar_accepted`, `collar_rejected`, `collar_broken`, `collar_lock_request`, `collar_locked`, `collar_unlocked`                                                                                                                                                                   |

Each `Notification` carries:

- `SourceCharacter` / `TargetCharacter` — character info with name, avatar, and user DID
- `Payload` — JSON with type-specific fields (parsed via `ParsePayload`)
- `Universe` — optional RP universe name
- `IsRead` / `ReadAt`

### API endpoints

| Method | Path                            | Purpose                                |
| ------ | ------------------------------- | -------------------------------------- |
| GET    | `/api/game/notifications`       | Paginated notifications (cursor-based) |
| GET    | `/api/game/notifications/count` | Unread count                           |
| POST   | `/api/game/notifications/read`  | Mark IDs as read                       |

---

## Interaction payloads

Key payload shapes for interactions:

- **Proposal**: predator/prey character names, path name, estimated duration, `initiatedBy`
- **Completed**: final node name, verb (past tense), `hasPointOfNoReturn`
- **Counter-proposal**: `counterCount`, flags for what changed (respawn time, branches, universe)
- **Safeword**: only path name and universe (source character omitted to protect who invoked it)
