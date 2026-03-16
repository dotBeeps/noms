# Themes

## Changing the theme

Set `name` under `[theme]` in `~/.config/noms/config.toml`:

```toml
[theme]
name = "dracula"
```

Changes take effect on next launch.

---

## Available themes

| Name           | Description                                                     |
| -------------- | --------------------------------------------------------------- |
| `default`      | Dark, muted purple/pink accent (ANSI 256-color)                 |
| `terminal`     | Pure ANSI 16-color — adapts to your terminal's own color scheme |
| `dracula`      | Classic Dracula: purple background, pink/cyan accents           |
| `nord`         | Arctic Nord: desaturated blue-grey                              |
| `tokyo-night`  | Deep navy blue with purple/blue accents                         |
| `rose-pine`    | Warm rose and pine: muted mauve/pink                            |
| `forest-night` | Earthy green/brown                                              |
| `neon-ember`   | Orange neon on very dark background                             |
| `retro-amber`  | Amber/gold terminal aesthetic                                   |
| `iceberg`      | Cool blue-grey, low contrast                                    |
| `mint-latte`   | Soft mint and cream                                             |

---

## Aliases

Many alternate spellings and aliases are accepted (case-insensitive, underscores/hyphens interchangeable):

| Alias                                                  | Resolves to    |
| ------------------------------------------------------ | -------------- |
| `tokyonight`, `tokyo_night`                            | `tokyo-night`  |
| `rosepine`, `rose_pine`                                | `rose-pine`    |
| `forestnight`, `forest_night`                          | `forest-night` |
| `neonember`, `neon_ember`                              | `neon-ember`   |
| `retroamber`, `retro_amber`                            | `retro-amber`  |
| `mintlatte`, `mint_latte`                              | `mint-latte`   |
| `iceberg-terminal`, `iceberg_terminal`                 | `iceberg`      |
| `nordic`                                               | `nord`         |
| `term`, `ansi`, `ansi-terminal`, `terminal-colors`     | `terminal`     |
| `dracula-terminal`, `dracula_terminal`                 | `dracula`      |
| `default-dark`, `default_terminal`, `default-terminal` | `default`      |

Unknown names fall back silently to `default`.

---

## Color palette roles

Each theme defines 18 semantic color roles. All UI components reference these roles rather than hard-coded colors.

| Role         | Used for                                                 |
| ------------ | -------------------------------------------------------- |
| `Primary`    | Tab bar active, headers, key UI elements                 |
| `Secondary`  | Inactive tabs, secondary text                            |
| `Accent`     | Selected items, highlights, like/repost indicators       |
| `Error`      | Error messages, destructive actions                      |
| `Success`    | Success confirmations                                    |
| `Muted`      | Timestamps, metadata, de-emphasized text                 |
| `Highlight`  | Highlighted text in search results                       |
| `Text`       | Body text                                                |
| `TextStrong` | Bold/important text                                      |
| `Surface`    | Panel backgrounds                                        |
| `SurfaceAlt` | Alternate surface (tab bar background, alternating rows) |
| `Border`     | Panel borders                                            |
| `Mention`    | @mention highlighting in post text                       |
| `Link`       | URL highlighting in post text                            |
| `Tag`        | #hashtag highlighting in post text                       |
| `Warning`    | Warning messages, rate limit notices                     |
| `OnPrimary`  | Text rendered on top of Primary-colored backgrounds      |
| `OnAccent`   | Text rendered on top of Accent-colored backgrounds       |

All colors are ANSI 256-color codes. The `terminal` theme uses ANSI 16-color codes (0–15) to inherit the terminal's own palette.

---

## How Apply() works

`theme.Apply(name)` in `internal/ui/theme/theme.go`:

1. Looks up the palette by name (resolving aliases)
2. Sets all 18 package-level `Color*` vars to `lipgloss.Color(code)`
3. Returns the resolved name

Style factory functions (`StylePost()`, `StyleHeader()`, etc.) construct fresh `lipgloss.Style` values on each call, so they always reflect the active theme even if `Apply` is called after init.
