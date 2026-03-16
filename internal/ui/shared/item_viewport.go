package shared

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
)

// PaginationThreshold is the number of items from the bottom at which
// we trigger loading the next page.
const PaginationThreshold = 3

// ItemViewport wraps viewport.Model with item-based selection and scrolling.
// It maintains a mapping from item indices to line offsets so that the
// line-based viewport can scroll to keep the selected item visible.
type ItemViewport struct {
	vp viewport.Model

	selectedIndex int
	itemCount     int
	width         int // logical width (for callers); viewport width is larger

	// lineOffsets[i] = starting line number of item i.
	// lineOffsets[itemCount] = total line count (sentinel).
	lineOffsets []int
}

// NewItemViewport creates an ItemViewport with the given dimensions.
// Width is stored for callers but does NOT constrain viewport rendering —
// content is pre-formatted by RenderItemWithBorder and may include border
// characters that exceed the logical width.
func NewItemViewport(width, height int) ItemViewport {
	// Use a large width so viewport never clips pre-formatted content
	// horizontally. Our items are already rendered to the correct visual width.
	vp := viewport.New(viewport.WithWidth(width+20), viewport.WithHeight(height))
	vp.MouseWheelEnabled = false  // we handle mouse wheel at the item level
	vp.KeyMap = viewport.KeyMap{} // we handle all keys ourselves
	return ItemViewport{vp: vp, width: width}
}

// SetSize updates the viewport dimensions.
func (iv *ItemViewport) SetSize(width, height int) {
	iv.width = width
	iv.vp.SetWidth(width + 20) // leave room for border chars / ANSI
	iv.vp.SetHeight(height)
}

// Width returns the logical width (not the internal viewport width, which is larger).
func (iv *ItemViewport) Width() int { return iv.width }

// Height returns the viewport height.
func (iv *ItemViewport) Height() int { return iv.vp.Height() }

// SetItems rebuilds the viewport content. renderFn is called for each item
// with the index and whether that item is currently selected.
func (iv *ItemViewport) SetItems(count int, renderFn func(index int, selected bool) string) {
	iv.itemCount = count
	if count == 0 {
		iv.selectedIndex = 0
	} else if iv.selectedIndex >= count {
		iv.selectedIndex = count - 1
	}

	// Build offsets into a local slice so IsNearVisible can still read the
	// previous (fully-computed) lineOffsets during the render loop.
	newOffsets := make([]int, count+1)
	var buf strings.Builder
	lineNum := 0
	for i := range count {
		newOffsets[i] = lineNum
		rendered := renderFn(i, i == iv.selectedIndex)
		buf.WriteString(rendered)
		lineNum += strings.Count(rendered, "\n")
	}
	newOffsets[count] = lineNum
	iv.lineOffsets = newOffsets

	iv.vp.SetContent(buf.String())
	iv.ensureVisible()
}

// SelectedIndex returns the current selection.
func (iv *ItemViewport) SelectedIndex() int { return iv.selectedIndex }

// SetSelectedIndex jumps to a specific index, clamped to bounds.
func (iv *ItemViewport) SetSelectedIndex(index int) {
	iv.selectedIndex = clamp(index, 0, max(0, iv.itemCount-1))
}

// MoveDown moves selection down by one. Returns true if selection changed.
func (iv *ItemViewport) MoveDown() bool {
	if iv.selectedIndex >= iv.itemCount-1 {
		return false
	}
	iv.selectedIndex++
	return true
}

// MoveUp moves selection up by one. Returns true if selection changed.
func (iv *ItemViewport) MoveUp() bool {
	if iv.selectedIndex <= 0 {
		return false
	}
	iv.selectedIndex--
	return true
}

// MoveDownN moves selection down by n items, clamped. Returns true if changed.
func (iv *ItemViewport) MoveDownN(n int) bool {
	old := iv.selectedIndex
	iv.selectedIndex = min(iv.selectedIndex+n, max(0, iv.itemCount-1))
	return iv.selectedIndex != old
}

// MoveUpN moves selection up by n items, clamped. Returns true if changed.
func (iv *ItemViewport) MoveUpN(n int) bool {
	old := iv.selectedIndex
	iv.selectedIndex = max(iv.selectedIndex-n, 0)
	return iv.selectedIndex != old
}

// MoveToTop moves selection to the first item.
func (iv *ItemViewport) MoveToTop() {
	iv.selectedIndex = 0
}

// MoveToBottom moves selection to the last item.
func (iv *ItemViewport) MoveToBottom() {
	iv.selectedIndex = max(0, iv.itemCount-1)
}

// NearBottom returns true when selectedIndex >= itemCount - threshold.
func (iv *ItemViewport) NearBottom(threshold int) bool {
	return iv.itemCount > 0 && iv.selectedIndex >= iv.itemCount-threshold
}

// AtTop returns true when the viewport scroll is at the top.
func (iv *ItemViewport) AtTop() bool { return iv.vp.AtTop() }

// AtBottom returns true when the viewport scroll is at the bottom.
func (iv *ItemViewport) AtBottom() bool { return iv.vp.AtBottom() }

// View returns the visible portion of the content.
func (iv *ItemViewport) View() string { return iv.vp.View() }

// Reset clears selection and scroll to zero.
func (iv *ItemViewport) Reset() {
	iv.selectedIndex = 0
	iv.vp.GotoTop()
}

// IsNearVisible returns whether item at index is within buffer lines of
// the current viewport scroll position. Returns true when unknown (first
// render or out-of-bounds index), so items are treated as visible by default.
func (iv *ItemViewport) IsNearVisible(index, buffer int) bool {
	if iv.lineOffsets == nil || index+1 >= len(iv.lineOffsets) {
		return true
	}
	itemStart := iv.lineOffsets[index]
	itemEnd := iv.lineOffsets[index+1]
	yOffset := iv.vp.YOffset()
	return itemEnd > yOffset-buffer && itemStart < yOffset+iv.vp.Height()+buffer
}

// ensureVisible scrolls the viewport so the selected item is visible.
func (iv *ItemViewport) ensureVisible() {
	if iv.itemCount == 0 || iv.lineOffsets == nil || len(iv.lineOffsets) <= iv.selectedIndex+1 {
		return
	}

	startLine := iv.lineOffsets[iv.selectedIndex]
	// endLine is the last line of this item (exclusive sentinel minus 1).
	endLine := iv.lineOffsets[iv.selectedIndex+1] - 1

	currentOffset := iv.vp.YOffset()
	visibleEnd := currentOffset + iv.vp.Height() - 1

	// Scroll down if selected item's bottom is below viewport.
	if endLine > visibleEnd {
		iv.vp.SetYOffset(endLine - iv.vp.Height() + 1)
	}
	// Scroll up if selected item's top is above viewport.
	if startLine < iv.vp.YOffset() {
		iv.vp.SetYOffset(startLine)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
