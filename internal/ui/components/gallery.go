package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

type GalleryImage struct {
	URL          string // fullsize URL (preferred)
	ThumbURL     string // thumbnail fallback (pre-cached from feed)
	AltText      string
	AspectWidth  int64
	AspectHeight int64
}

type galleryKeys struct {
	Next  key.Binding
	Prev  key.Binding
	Close key.Binding
}

var galleryKeyMap = galleryKeys{
	Next:  key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "next")),
	Prev:  key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "prev")),
	Close: key.NewBinding(key.WithKeys("esc", "i"), key.WithHelp("esc/i", "close")),
}

type GalleryModel struct {
	Images  []GalleryImage
	Current int
	Width   int
	Height  int
	Visible bool
	cache   images.ImageRenderer
}

func NewGalleryModel(cache images.ImageRenderer) GalleryModel {
	return GalleryModel{cache: cache}
}

func (m *GalleryModel) Open(imgs []GalleryImage, startIndex int) {
	m.Images = imgs
	m.Current = startIndex
	m.Visible = true
}

func (m *GalleryModel) Close() {
	m.Visible = false
}

func (m GalleryModel) Update(msg tea.Msg) (GalleryModel, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(kp, galleryKeyMap.Next):
		if m.Current < len(m.Images)-1 {
			m.Current++
		}
	case key.Matches(kp, galleryKeyMap.Prev):
		if m.Current > 0 {
			m.Current--
		}
	case key.Matches(kp, galleryKeyMap.Close):
		m.Visible = false
	}
	return m, nil
}

func (m GalleryModel) View() string {
	if !m.Visible || len(m.Images) == 0 {
		return ""
	}

	img := m.Images[m.Current]
	muted := theme.StyleMuted()

	// Fixed chrome: counter (1) + alt (1) + hints (1) = 3 rows
	const chromeRows = 3
	imgHeight := m.Height - chromeRows
	if imgHeight < 4 {
		imgHeight = 4
	}
	imgWidth := m.Width
	if imgWidth < 10 {
		imgWidth = 10
	}

	// Counter
	counter := fmt.Sprintf("%d / %d", m.Current+1, len(m.Images))
	counterLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, muted.Render(counter))

	// Fit to window using API aspect ratio, fallback to cached dims
	renderCols, renderRows := fitDims(img, imgWidth, imgHeight, m.cache)

	// Prefer fullsize, fall back to thumb while it loads
	renderURL := img.URL
	if m.cache == nil || !m.cache.Enabled() || !m.cache.IsCached(renderURL) {
		renderURL = img.ThumbURL
	}

	var imgStr string
	if m.cache != nil && m.cache.Enabled() && renderURL != "" && m.cache.IsCached(renderURL) {
		imgStr = strings.TrimRight(m.cache.RenderImage(renderURL, renderCols, renderRows), "\n ")
	}
	if imgStr == "" {
		imgStr = lipgloss.NewStyle().
			Width(renderCols).
			Height(renderRows).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(theme.ColorMuted).
			Render("[ loading ]")
	}

	// Manually center each image line (safe for Kitty placeholder chars)
	leftPad := (m.Width - renderCols) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	hpad := strings.Repeat(" ", leftPad)
	imgLines := strings.Split(imgStr, "\n")
	for i, line := range imgLines {
		imgLines[i] = hpad + line
	}

	// Alt text
	var altLine string
	if img.AltText != "" {
		altLine = lipgloss.PlaceHorizontal(m.Width, lipgloss.Center,
			muted.Render(shared.TruncateStr(img.AltText, m.Width-4)))
	}

	// Nav hints
	var hintsText string
	if len(m.Images) > 1 {
		hintsText = "h/← prev  •  l/→ next  •  esc/i close"
	} else {
		hintsText = "esc/i close"
	}
	hintsLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, muted.Render(hintsText))

	// Assemble content lines (image block + chrome)
	contentLines := 1 + len(imgLines) + 1 // counter + image + hints
	if altLine != "" {
		contentLines++
	}

	// Vertically center in the available height by padding top and bottom
	totalHeight := m.Height
	topPad := (totalHeight - contentLines) / 2
	if topPad < 0 {
		topPad = 0
	}
	bottomPad := totalHeight - topPad - contentLines
	if bottomPad < 0 {
		bottomPad = 0
	}

	blank := strings.Repeat(" ", m.Width)
	var b strings.Builder
	for range topPad {
		b.WriteString(blank)
		b.WriteByte('\n')
	}
	b.WriteString(counterLine)
	b.WriteByte('\n')
	b.WriteString(strings.Join(imgLines, "\n"))
	b.WriteByte('\n')
	if altLine != "" {
		b.WriteString(altLine)
		b.WriteByte('\n')
	}
	b.WriteString(hintsLine)
	for range bottomPad {
		b.WriteByte('\n')
		b.WriteString(blank)
	}
	return b.String()
}

// fitDims calculates terminal cols×rows for an image that fits within maxCols×maxRows,
// preserving aspect ratio. Uses AspectRatio from the API if available, otherwise
// falls back to the pixel dimensions stored in the image cache.
func fitDims(img GalleryImage, maxCols, maxRows int, cache images.ImageRenderer) (cols, rows int) {
	var pixW, pixH int

	if img.AspectWidth > 0 && img.AspectHeight > 0 {
		pixW, pixH = int(img.AspectWidth), int(img.AspectHeight)
	} else if cache != nil {
		url := img.URL
		if url == "" {
			url = img.ThumbURL
		}
		if w, h, ok := cache.Dimensions(url); ok && w > 0 {
			pixW, pixH = w, h
		}
	}

	if pixW == 0 || pixH == 0 {
		return maxCols, maxRows
	}

	// Terminal cells are ~2x taller than wide, so divide pixel height by 2
	// when mapping to rows.
	cols = maxCols
	rows = maxCols * pixH / (pixW * 2)
	if rows > maxRows {
		rows = maxRows
		cols = maxRows * pixW * 2 / pixH
		if cols > maxCols {
			cols = maxCols
		}
	}
	if rows < 4 {
		rows = 4
	}
	if cols < 4 {
		cols = 4
	}
	return cols, rows
}
