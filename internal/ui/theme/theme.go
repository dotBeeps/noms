package theme

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"charm.land/lipgloss/v2"
)

// ColorPair holds dark and light variants of a color (ANSI 256 code strings).
// When both values are the same, the color is mode-invariant.
type ColorPair struct {
	Dark  string
	Light string
}

// C is a shorthand constructor for a dark-only ColorPair (light = dark).
func C(code string) ColorPair {
	return ColorPair{Dark: code, Light: code}
}

// DL constructs a ColorPair with distinct dark and light values.
func DL(dark, light string) ColorPair {
	return ColorPair{Dark: dark, Light: light}
}

type Palette struct {
	Name       string
	Primary    ColorPair
	Secondary  ColorPair
	Accent     ColorPair
	Error      ColorPair
	Success    ColorPair
	Muted      ColorPair
	Highlight  ColorPair
	Text       ColorPair
	TextStrong ColorPair
	Surface    ColorPair
	SurfaceAlt ColorPair
	Border     ColorPair
	Mention    ColorPair
	Link       ColorPair
	Tag        ColorPair
	Warning    ColorPair
	OnPrimary  ColorPair
	OnAccent   ColorPair
}

var palettes = map[string]Palette{
	"default": {
		Name:       "default",
		Primary:    DL("62", "61"),
		Secondary:  DL("243", "247"),
		Accent:     DL("205", "162"),
		Error:      DL("196", "160"),
		Success:    DL("78", "28"),
		Muted:      DL("241", "249"),
		Highlight:  DL("229", "94"),
		Text:       DL("252", "235"),
		TextStrong: DL("255", "232"),
		Surface:    DL("236", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("240", "250"),
		Mention:    DL("33", "25"),
		Link:       DL("45", "27"),
		Tag:        DL("141", "97"),
		Warning:    DL("203", "166"),
		OnPrimary:  DL("229", "255"),
		OnAccent:   DL("255", "255"),
	},
	"terminal": {
		Name:       "terminal",
		Primary:    C("4"),
		Secondary:  C("8"),
		Accent:     C("6"),
		Error:      C("1"),
		Success:    C("2"),
		Muted:      C("8"),
		Highlight:  C("11"),
		Text:       C("7"),
		TextStrong: C("15"),
		Surface:    C("0"),
		SurfaceAlt: C("8"),
		Border:     C("8"),
		Mention:    C("14"),
		Link:       C("12"),
		Tag:        C("13"),
		Warning:    C("3"),
		OnPrimary:  C("15"),
		OnAccent:   C("0"),
	},
	"dracula": {
		Name:       "dracula",
		Primary:    DL("141", "98"),
		Secondary:  DL("240", "248"),
		Accent:     DL("212", "162"),
		Error:      DL("203", "160"),
		Success:    DL("84", "28"),
		Muted:      DL("245", "249"),
		Highlight:  DL("228", "94"),
		Text:       DL("253", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("236", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("61", "146"),
		Mention:    DL("117", "25"),
		Link:       DL("81", "27"),
		Tag:        DL("176", "133"),
		Warning:    DL("215", "172"),
		OnPrimary:  DL("255", "255"),
		OnAccent:   DL("236", "254"),
	},
	"nord": {
		Name:       "nord",
		Primary:    DL("110", "67"),
		Secondary:  DL("102", "248"),
		Accent:     DL("81", "31"),
		Error:      DL("203", "160"),
		Success:    DL("114", "28"),
		Muted:      DL("244", "249"),
		Highlight:  DL("223", "130"),
		Text:       DL("252", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("236", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("67", "110"),
		Mention:    DL("81", "31"),
		Link:       DL("110", "67"),
		Tag:        DL("180", "137"),
		Warning:    DL("215", "172"),
		OnPrimary:  DL("236", "255"),
		OnAccent:   DL("236", "255"),
	},
	"tokyo-night": {
		Name:       "tokyo-night",
		Primary:    DL("111", "62"),
		Secondary:  DL("60", "248"),
		Accent:     DL("183", "133"),
		Error:      DL("203", "160"),
		Success:    DL("114", "28"),
		Muted:      DL("146", "249"),
		Highlight:  DL("223", "130"),
		Text:       DL("252", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("235", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("60", "146"),
		Mention:    DL("117", "25"),
		Link:       DL("117", "25"),
		Tag:        DL("183", "133"),
		Warning:    DL("214", "172"),
		OnPrimary:  DL("235", "255"),
		OnAccent:   DL("235", "255"),
	},
	"rose-pine": {
		Name:       "rose-pine",
		Primary:    DL("181", "132"),
		Secondary:  DL("102", "248"),
		Accent:     DL("175", "132"),
		Error:      DL("174", "131"),
		Success:    DL("108", "65"),
		Muted:      DL("245", "249"),
		Highlight:  DL("223", "130"),
		Text:       DL("252", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("235", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("102", "181"),
		Mention:    DL("152", "67"),
		Link:       DL("153", "68"),
		Tag:        DL("181", "132"),
		Warning:    DL("216", "173"),
		OnPrimary:  DL("235", "255"),
		OnAccent:   DL("235", "255"),
	},
	"forest-night": {
		Name:       "forest-night",
		Primary:    DL("114", "28"),
		Secondary:  DL("65", "248"),
		Accent:     DL("150", "65"),
		Error:      DL("203", "160"),
		Success:    DL("108", "22"),
		Muted:      DL("242", "249"),
		Highlight:  DL("187", "94"),
		Text:       DL("252", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("235", "254"),
		SurfaceAlt: DL("236", "253"),
		Border:     DL("65", "114"),
		Mention:    DL("109", "30"),
		Link:       DL("115", "29"),
		Tag:        DL("151", "65"),
		Warning:    DL("179", "136"),
		OnPrimary:  DL("235", "255"),
		OnAccent:   DL("235", "255"),
	},
	"neon-ember": {
		Name:       "neon-ember",
		Primary:    DL("208", "166"),
		Secondary:  DL("240", "248"),
		Accent:     DL("198", "161"),
		Error:      DL("196", "160"),
		Success:    DL("118", "28"),
		Muted:      DL("246", "249"),
		Highlight:  DL("227", "130"),
		Text:       DL("252", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("234", "254"),
		SurfaceAlt: DL("236", "253"),
		Border:     DL("239", "250"),
		Mention:    DL("45", "27"),
		Link:       DL("51", "21"),
		Tag:        DL("177", "133"),
		Warning:    DL("208", "166"),
		OnPrimary:  DL("234", "255"),
		OnAccent:   DL("234", "255"),
	},
	"retro-amber": {
		Name:       "retro-amber",
		Primary:    DL("214", "130"),
		Secondary:  DL("136", "248"),
		Accent:     DL("221", "136"),
		Error:      DL("166", "124"),
		Success:    DL("178", "100"),
		Muted:      DL("243", "249"),
		Highlight:  DL("230", "94"),
		Text:       DL("223", "236"),
		TextStrong: DL("230", "232"),
		Surface:    DL("235", "254"),
		SurfaceAlt: DL("237", "253"),
		Border:     DL("137", "179"),
		Mention:    DL("215", "130"),
		Link:       DL("221", "136"),
		Tag:        DL("180", "137"),
		Warning:    DL("214", "172"),
		OnPrimary:  DL("234", "255"),
		OnAccent:   DL("235", "255"),
	},
	"iceberg": {
		Name:       "iceberg",
		Primary:    DL("110", "67"),
		Secondary:  DL("66", "248"),
		Accent:     DL("117", "31"),
		Error:      DL("204", "161"),
		Success:    DL("114", "28"),
		Muted:      DL("246", "249"),
		Highlight:  DL("153", "68"),
		Text:       DL("254", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("235", "254"),
		SurfaceAlt: DL("236", "253"),
		Border:     DL("67", "110"),
		Mention:    DL("117", "25"),
		Link:       DL("153", "68"),
		Tag:        DL("146", "97"),
		Warning:    DL("216", "173"),
		OnPrimary:  DL("235", "255"),
		OnAccent:   DL("235", "255"),
	},
	"mint-latte": {
		Name:       "mint-latte",
		Primary:    DL("78", "29"),
		Secondary:  DL("145", "248"),
		Accent:     DL("151", "65"),
		Error:      DL("203", "160"),
		Success:    DL("71", "22"),
		Muted:      DL("248", "243"),
		Highlight:  DL("229", "94"),
		Text:       DL("254", "236"),
		TextStrong: DL("255", "232"),
		Surface:    DL("237", "254"),
		SurfaceAlt: DL("238", "253"),
		Border:     DL("109", "151"),
		Mention:    DL("79", "29"),
		Link:       DL("116", "30"),
		Tag:        DL("150", "65"),
		Warning:    DL("215", "172"),
		OnPrimary:  DL("235", "255"),
		OnAccent:   DL("235", "255"),
	},
}

var paletteAliases = map[string]string{
	"tokyonight":       "tokyo-night",
	"rosepine":         "rose-pine",
	"forestnight":      "forest-night",
	"neonember":        "neon-ember",
	"retroamber":       "retro-amber",
	"iceberg-terminal": "iceberg",
	"mintlatte":        "mint-latte",
	"default-dark":     "default",
	"default_terminal": "default",
	"default-terminal": "default",
	"nordic":           "nord",
	"dracula-terminal": "dracula",
	"tokyo_night":      "tokyo-night",
	"rose_pine":        "rose-pine",
	"forest_night":     "forest-night",
	"neon_ember":       "neon-ember",
	"retro_amber":      "retro-amber",
	"mint_latte":       "mint-latte",
	"iceberg_terminal": "iceberg",
	"dracula_terminal": "dracula",
	"term":             "terminal",
	"ansi":             "terminal",
	"ansi-terminal":    "terminal",
	"term-colors":      "terminal",
	"terminal-colors":  "terminal",
}

var activePalette = palettes["default"]
var isDark = true // assume dark until terminal reports otherwise
var applyMu sync.Mutex
var currentThemeKey atomic.Value // stores string "name:dark|light"; set by Apply

// Resolved surface codes for ANSI escape sequences.
var resolvedSurfaceCode string
var resolvedSurfaceAltCode string

// Color palette constants
var (
	ColorPrimary    = lipgloss.Color("62")
	ColorSecondary  = lipgloss.Color("243")
	ColorAccent     = lipgloss.Color("205")
	ColorError      = lipgloss.Color("196")
	ColorSuccess    = lipgloss.Color("78")
	ColorMuted      = lipgloss.Color("241")
	ColorHighlight  = lipgloss.Color("229")
	ColorText       = lipgloss.Color("252")
	ColorTextStrong = lipgloss.Color("255")
	ColorSurface    = lipgloss.Color("236")
	ColorSurfaceAlt = lipgloss.Color("237")
	ColorBorder     = lipgloss.Color("240")
	ColorMention    = lipgloss.Color("33")
	ColorLink       = lipgloss.Color("45")
	ColorTag        = lipgloss.Color("141")
	ColorWarning    = lipgloss.Color("203")
	ColorOnPrimary  = lipgloss.Color("229")
	ColorOnAccent   = lipgloss.Color("255")
)

// Layout constants
const (
	// TabBarHeight is the height of the tab bar
	TabBarHeight = 1
	// StatusBarHeight is the height of the status bar
	StatusBarHeight = 1
)

// Style factory functions -- constructed on call so they always reflect the active theme.

func StylePost() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorText)
}

func StyleHeader() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
}

func StyleSelected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
}

func StyleMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorMuted)
}

func StyleError() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorError)
}

func StyleWarning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorWarning)
}

func StyleHeaderSubtle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorPrimary)
}

func StyleTabActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorAccent).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorAccent).
		Padding(0, 1)
}

func StyleTabInactive() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorSecondary).Padding(0, 1)
}

func init() {
	Apply("default")
}

func AvailableThemes() []string {
	names := make([]string, 0, len(palettes))
	for name := range palettes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ActiveTheme() string {
	return activePalette.Name
}

// IsDark reports whether the terminal has a dark background.
func IsDark() bool {
	return isDark
}

// SetDarkMode stores the terminal background darkness and re-applies the
// active theme so that LightDark-aware colors are updated.
func SetDarkMode(dark bool) string {
	isDark = dark
	return Apply(activePalette.Name)
}

func SurfaceCode() string {
	return resolvedSurfaceCode
}

func SurfaceAltCode() string {
	return resolvedSurfaceAltCode
}

// themeKey returns a cache key that encodes both theme name and dark/light mode.
func themeKey(name string) string {
	if isDark {
		return name + ":dark"
	}
	return name + ":light"
}

// resolve picks the dark or light value from a ColorPair based on isDark.
func resolve(pair ColorPair) string {
	if isDark {
		return pair.Dark
	}
	return pair.Light
}

func Apply(name string) string {
	p, ok := lookupPalette(name)
	if !ok {
		p = palettes["default"]
	}

	key := themeKey(p.Name)

	// Fast path: skip all writes if theme + mode haven't changed.
	// This prevents data races when parallel tests all call Apply("default")
	// after init() has already applied it.
	if cur, _ := currentThemeKey.Load().(string); cur == key {
		return p.Name
	}

	applyMu.Lock()
	defer applyMu.Unlock()

	// Re-check under write lock.
	if cur, _ := currentThemeKey.Load().(string); cur == key {
		return p.Name
	}

	activePalette = p

	pick := lipgloss.LightDark(isDark)

	ColorPrimary = pick(lipgloss.Color(p.Primary.Light), lipgloss.Color(p.Primary.Dark))
	ColorSecondary = pick(lipgloss.Color(p.Secondary.Light), lipgloss.Color(p.Secondary.Dark))
	ColorAccent = pick(lipgloss.Color(p.Accent.Light), lipgloss.Color(p.Accent.Dark))
	ColorError = pick(lipgloss.Color(p.Error.Light), lipgloss.Color(p.Error.Dark))
	ColorSuccess = pick(lipgloss.Color(p.Success.Light), lipgloss.Color(p.Success.Dark))
	ColorMuted = pick(lipgloss.Color(p.Muted.Light), lipgloss.Color(p.Muted.Dark))
	ColorHighlight = pick(lipgloss.Color(p.Highlight.Light), lipgloss.Color(p.Highlight.Dark))
	ColorText = pick(lipgloss.Color(p.Text.Light), lipgloss.Color(p.Text.Dark))
	ColorTextStrong = pick(lipgloss.Color(p.TextStrong.Light), lipgloss.Color(p.TextStrong.Dark))
	ColorSurface = pick(lipgloss.Color(p.Surface.Light), lipgloss.Color(p.Surface.Dark))
	ColorSurfaceAlt = pick(lipgloss.Color(p.SurfaceAlt.Light), lipgloss.Color(p.SurfaceAlt.Dark))
	ColorBorder = pick(lipgloss.Color(p.Border.Light), lipgloss.Color(p.Border.Dark))
	ColorMention = pick(lipgloss.Color(p.Mention.Light), lipgloss.Color(p.Mention.Dark))
	ColorLink = pick(lipgloss.Color(p.Link.Light), lipgloss.Color(p.Link.Dark))
	ColorTag = pick(lipgloss.Color(p.Tag.Light), lipgloss.Color(p.Tag.Dark))
	ColorWarning = pick(lipgloss.Color(p.Warning.Light), lipgloss.Color(p.Warning.Dark))
	ColorOnPrimary = pick(lipgloss.Color(p.OnPrimary.Light), lipgloss.Color(p.OnPrimary.Dark))
	ColorOnAccent = pick(lipgloss.Color(p.OnAccent.Light), lipgloss.Color(p.OnAccent.Dark))

	resolvedSurfaceCode = resolve(p.Surface)
	resolvedSurfaceAltCode = resolve(p.SurfaceAlt)

	currentThemeKey.Store(key)
	return activePalette.Name
}

func lookupPalette(name string) (Palette, bool) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if alias, ok := paletteAliases[normalized]; ok {
		normalized = alias
	}
	p, ok := palettes[normalized]
	return p, ok
}
