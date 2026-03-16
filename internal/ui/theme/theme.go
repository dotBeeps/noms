package theme

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

type Palette struct {
	Name       string
	Primary    string
	Secondary  string
	Accent     string
	Error      string
	Success    string
	Muted      string
	Highlight  string
	Text       string
	TextStrong string
	Surface    string
	SurfaceAlt string
	Border     string
	Mention    string
	Link       string
	Tag        string
	Warning    string
	OnPrimary  string
	OnAccent   string
}

var palettes = map[string]Palette{
	"default": {
		Name:       "default",
		Primary:    "62",
		Secondary:  "243",
		Accent:     "205",
		Error:      "196",
		Success:    "78",
		Muted:      "241",
		Highlight:  "229",
		Text:       "252",
		TextStrong: "255",
		Surface:    "236",
		SurfaceAlt: "237",
		Border:     "240",
		Mention:    "33",
		Link:       "45",
		Tag:        "141",
		Warning:    "203",
		OnPrimary:  "229",
		OnAccent:   "255",
	},
	"terminal": {
		Name:       "terminal",
		Primary:    "4",
		Secondary:  "8",
		Accent:     "6",
		Error:      "1",
		Success:    "2",
		Muted:      "8",
		Highlight:  "11",
		Text:       "7",
		TextStrong: "15",
		Surface:    "0",
		SurfaceAlt: "8",
		Border:     "8",
		Mention:    "14",
		Link:       "12",
		Tag:        "13",
		Warning:    "3",
		OnPrimary:  "15",
		OnAccent:   "0",
	},
	"dracula": {
		Name:       "dracula",
		Primary:    "141",
		Secondary:  "240",
		Accent:     "212",
		Error:      "203",
		Success:    "84",
		Muted:      "245",
		Highlight:  "228",
		Text:       "253",
		TextStrong: "255",
		Surface:    "236",
		SurfaceAlt: "237",
		Border:     "61",
		Mention:    "117",
		Link:       "81",
		Tag:        "176",
		Warning:    "215",
		OnPrimary:  "255",
		OnAccent:   "236",
	},
	"nord": {
		Name:       "nord",
		Primary:    "110",
		Secondary:  "102",
		Accent:     "81",
		Error:      "203",
		Success:    "114",
		Muted:      "244",
		Highlight:  "223",
		Text:       "252",
		TextStrong: "255",
		Surface:    "236",
		SurfaceAlt: "237",
		Border:     "67",
		Mention:    "81",
		Link:       "110",
		Tag:        "180",
		Warning:    "215",
		OnPrimary:  "236",
		OnAccent:   "236",
	},
	"tokyo-night": {
		Name:       "tokyo-night",
		Primary:    "111",
		Secondary:  "60",
		Accent:     "183",
		Error:      "203",
		Success:    "114",
		Muted:      "146",
		Highlight:  "223",
		Text:       "252",
		TextStrong: "255",
		Surface:    "235",
		SurfaceAlt: "237",
		Border:     "60",
		Mention:    "117",
		Link:       "117",
		Tag:        "183",
		Warning:    "214",
		OnPrimary:  "235",
		OnAccent:   "235",
	},
	"rose-pine": {
		Name:       "rose-pine",
		Primary:    "181",
		Secondary:  "102",
		Accent:     "175",
		Error:      "174",
		Success:    "108",
		Muted:      "245",
		Highlight:  "223",
		Text:       "252",
		TextStrong: "255",
		Surface:    "235",
		SurfaceAlt: "237",
		Border:     "102",
		Mention:    "152",
		Link:       "153",
		Tag:        "181",
		Warning:    "216",
		OnPrimary:  "235",
		OnAccent:   "235",
	},
	"forest-night": {
		Name:       "forest-night",
		Primary:    "114",
		Secondary:  "65",
		Accent:     "150",
		Error:      "203",
		Success:    "108",
		Muted:      "242",
		Highlight:  "187",
		Text:       "252",
		TextStrong: "255",
		Surface:    "235",
		SurfaceAlt: "236",
		Border:     "65",
		Mention:    "109",
		Link:       "115",
		Tag:        "151",
		Warning:    "179",
		OnPrimary:  "235",
		OnAccent:   "235",
	},
	"neon-ember": {
		Name:       "neon-ember",
		Primary:    "208",
		Secondary:  "240",
		Accent:     "198",
		Error:      "196",
		Success:    "118",
		Muted:      "246",
		Highlight:  "227",
		Text:       "252",
		TextStrong: "255",
		Surface:    "234",
		SurfaceAlt: "236",
		Border:     "239",
		Mention:    "45",
		Link:       "51",
		Tag:        "177",
		Warning:    "208",
		OnPrimary:  "234",
		OnAccent:   "234",
	},
	"retro-amber": {
		Name:       "retro-amber",
		Primary:    "214",
		Secondary:  "136",
		Accent:     "221",
		Error:      "166",
		Success:    "178",
		Muted:      "243",
		Highlight:  "230",
		Text:       "223",
		TextStrong: "230",
		Surface:    "235",
		SurfaceAlt: "237",
		Border:     "137",
		Mention:    "215",
		Link:       "221",
		Tag:        "180",
		Warning:    "214",
		OnPrimary:  "234",
		OnAccent:   "235",
	},
	"iceberg": {
		Name:       "iceberg",
		Primary:    "110",
		Secondary:  "66",
		Accent:     "117",
		Error:      "204",
		Success:    "114",
		Muted:      "246",
		Highlight:  "153",
		Text:       "254",
		TextStrong: "255",
		Surface:    "235",
		SurfaceAlt: "236",
		Border:     "67",
		Mention:    "117",
		Link:       "153",
		Tag:        "146",
		Warning:    "216",
		OnPrimary:  "235",
		OnAccent:   "235",
	},
	"mint-latte": {
		Name:       "mint-latte",
		Primary:    "78",
		Secondary:  "145",
		Accent:     "151",
		Error:      "203",
		Success:    "71",
		Muted:      "248",
		Highlight:  "229",
		Text:       "254",
		TextStrong: "255",
		Surface:    "237",
		SurfaceAlt: "238",
		Border:     "109",
		Mention:    "79",
		Link:       "116",
		Tag:        "150",
		Warning:    "215",
		OnPrimary:  "235",
		OnAccent:   "235",
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

// Style factory functions — constructed on call so they always reflect the active theme.

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
// active theme so that any LightDark-aware colors are updated.
func SetDarkMode(dark bool) string {
	isDark = dark
	return Apply(activePalette.Name)
}

func SurfaceCode() string {
	return activePalette.Surface
}

func SurfaceAltCode() string {
	return activePalette.SurfaceAlt
}

func Apply(name string) string {
	p, ok := lookupPalette(name)
	if !ok {
		p = palettes["default"]
	}

	activePalette = p

	ColorPrimary = lipgloss.Color(p.Primary)
	ColorSecondary = lipgloss.Color(p.Secondary)
	ColorAccent = lipgloss.Color(p.Accent)
	ColorError = lipgloss.Color(p.Error)
	ColorSuccess = lipgloss.Color(p.Success)
	ColorMuted = lipgloss.Color(p.Muted)
	ColorHighlight = lipgloss.Color(p.Highlight)
	ColorText = lipgloss.Color(p.Text)
	ColorTextStrong = lipgloss.Color(p.TextStrong)
	ColorSurface = lipgloss.Color(p.Surface)
	ColorSurfaceAlt = lipgloss.Color(p.SurfaceAlt)
	ColorBorder = lipgloss.Color(p.Border)
	ColorMention = lipgloss.Color(p.Mention)
	ColorLink = lipgloss.Color(p.Link)
	ColorTag = lipgloss.Color(p.Tag)
	ColorWarning = lipgloss.Color(p.Warning)
	ColorOnPrimary = lipgloss.Color(p.OnPrimary)
	ColorOnAccent = lipgloss.Color(p.OnAccent)

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
