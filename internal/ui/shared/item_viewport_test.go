package shared

import (
	"fmt"
	"strings"
	"testing"
)

// simpleRender returns a render function that produces items of a fixed height.
func simpleRender(linesPerItem int) func(int, bool) string {
	return func(index int, selected bool) string {
		sel := " "
		if selected {
			sel = ">"
		}
		var b strings.Builder
		for j := range linesPerItem {
			fmt.Fprintf(&b, "%sitem %d line %d\n", sel, index, j)
		}
		return b.String()
	}
}

func TestItemViewport_BasicNavigation(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(5, simpleRender(2))

	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}

	if !iv.MoveDown() {
		t.Error("MoveDown should return true")
	}
	iv.SetItems(5, simpleRender(2)) // rebuild to reflect selection
	if iv.SelectedIndex() != 1 {
		t.Errorf("expected selected 1, got %d", iv.SelectedIndex())
	}

	if !iv.MoveUp() {
		t.Error("MoveUp should return true")
	}
	iv.SetItems(5, simpleRender(2))
	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}

	// Can't go above 0
	if iv.MoveUp() {
		t.Error("MoveUp at top should return false")
	}
}

func TestItemViewport_BoundsClamp(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(3, simpleRender(2))

	// Move to bottom
	iv.MoveDown()
	iv.MoveDown()
	if iv.SelectedIndex() != 2 {
		t.Errorf("expected selected 2, got %d", iv.SelectedIndex())
	}

	// Can't go past last item
	if iv.MoveDown() {
		t.Error("MoveDown at bottom should return false")
	}
	if iv.SelectedIndex() != 2 {
		t.Errorf("expected selected 2, got %d", iv.SelectedIndex())
	}
}

func TestItemViewport_MoveN(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 20)
	iv.SetItems(10, simpleRender(2))

	iv.MoveDownN(3)
	if iv.SelectedIndex() != 3 {
		t.Errorf("expected selected 3, got %d", iv.SelectedIndex())
	}

	iv.MoveUpN(2)
	if iv.SelectedIndex() != 1 {
		t.Errorf("expected selected 1, got %d", iv.SelectedIndex())
	}

	// Clamps to bounds
	iv.MoveUpN(100)
	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}

	iv.MoveDownN(100)
	if iv.SelectedIndex() != 9 {
		t.Errorf("expected selected 9, got %d", iv.SelectedIndex())
	}
}

func TestItemViewport_NearBottom(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 20)
	iv.SetItems(10, simpleRender(2))

	if iv.NearBottom(3) {
		t.Error("at index 0, NearBottom(3) should be false")
	}

	iv.MoveDownN(7)
	if !iv.NearBottom(3) {
		t.Errorf("at index 7 of 10, NearBottom(3) should be true")
	}

	iv.MoveDownN(2)
	if !iv.NearBottom(3) {
		t.Errorf("at index 9 of 10, NearBottom(3) should be true")
	}
}

func TestItemViewport_EmptyList(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(0, simpleRender(2))

	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}

	if iv.MoveDown() {
		t.Error("MoveDown on empty list should return false")
	}
	if iv.MoveUp() {
		t.Error("MoveUp on empty list should return false")
	}

	// View should not panic
	_ = iv.View()
}

// TestItemViewport_EmptyListAfterSelection verifies that SetItems(0) after a
// non-zero selection resets selectedIndex to 0 and leaves the list empty so
// navigation and NearBottom behave correctly.
func TestItemViewport_EmptyListAfterSelection(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(5, simpleRender(2))
	iv.MoveDownN(4) // selectedIndex = 4

	// Shrink to empty — selectedIndex must reset to 0 without panicking.
	iv.SetItems(0, simpleRender(2))

	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0 after SetItems(0), got %d", iv.SelectedIndex())
	}
	// itemCount must be 0 so callers know the list is empty.
	if iv.NearBottom(1) {
		t.Error("NearBottom should be false for empty list")
	}
	// View and navigation must not panic.
	_ = iv.View()
	if iv.MoveDown() {
		t.Error("MoveDown on empty list should return false")
	}
	if iv.MoveUp() {
		t.Error("MoveUp on empty list should return false")
	}
}

// TestItemViewport_NeverSetItems verifies that using an ItemViewport without
// ever calling SetItems does not panic — lineOffsets is nil until first use.
func TestItemViewport_NeverSetItems(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetSize(40, 10)              // must not panic
	iv.SetItems(0, simpleRender(2)) // must not panic (calls ensureVisible)
	_ = iv.View()
}

func TestItemViewport_SingleItem(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(1, simpleRender(2))

	if !iv.NearBottom(3) {
		t.Error("single item should be NearBottom(3)")
	}

	if iv.MoveDown() {
		t.Error("MoveDown on single item should return false")
	}
	if iv.MoveUp() {
		t.Error("MoveUp on single item should return false")
	}
}

func TestItemViewport_Reset(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(10, simpleRender(2))
	iv.MoveDownN(5)
	iv.SetItems(10, simpleRender(2))

	iv.Reset()
	if iv.SelectedIndex() != 0 {
		t.Errorf("after Reset, expected selected 0, got %d", iv.SelectedIndex())
	}
}

func TestItemViewport_SetSelectedIndex(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(10, simpleRender(2))

	iv.SetSelectedIndex(5)
	if iv.SelectedIndex() != 5 {
		t.Errorf("expected selected 5, got %d", iv.SelectedIndex())
	}

	// Clamps to bounds
	iv.SetSelectedIndex(100)
	if iv.SelectedIndex() != 9 {
		t.Errorf("expected selected 9, got %d", iv.SelectedIndex())
	}

	iv.SetSelectedIndex(-5)
	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}
}

func TestItemViewport_ScrollsToSelection(t *testing.T) {
	t.Parallel()

	// 10 items, 3 lines each = 30 lines total. Viewport = 9 lines high.
	// Only ~3 items fit at once.
	iv := NewItemViewport(40, 9)
	iv.SetItems(10, simpleRender(3))

	// At index 0, content should contain "item 0"
	view := iv.View()
	if !strings.Contains(view, "item 0") {
		t.Error("at index 0, view should contain item 0")
	}

	// Move to index 5 and rebuild — view should contain "item 5"
	iv.MoveDownN(5)
	iv.SetItems(10, simpleRender(3))
	view = iv.View()
	if !strings.Contains(view, "item 5") {
		t.Errorf("at index 5, view should contain item 5")
	}

	// Move back to 0 — should scroll up
	iv.MoveToTop()
	iv.SetItems(10, simpleRender(3))
	view = iv.View()
	if !strings.Contains(view, "item 0") {
		t.Error("after MoveToTop, view should contain item 0")
	}
}

func TestItemViewport_VariableHeight(t *testing.T) {
	t.Parallel()

	// Items with variable heights: item 0 = 1 line, item 1 = 5 lines, item 2 = 1 line
	heights := []int{1, 5, 1}
	render := func(index int, selected bool) string {
		var b strings.Builder
		for j := range heights[index] {
			fmt.Fprintf(&b, "item %d line %d\n", index, j)
		}
		return b.String()
	}

	iv := NewItemViewport(40, 4)
	iv.SetItems(3, render)

	// Move to item 1 (5 lines tall, taller than viewport)
	iv.MoveDown()
	iv.SetItems(3, render)
	view := iv.View()
	if !strings.Contains(view, "item 1") {
		t.Error("at index 1, view should contain item 1")
	}

	// Move to item 2 — should scroll past the tall item
	iv.MoveDown()
	iv.SetItems(3, render)
	view = iv.View()
	if !strings.Contains(view, "item 2") {
		t.Error("at index 2, view should contain item 2")
	}
}

func TestItemViewport_MoveToTopBottom(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(20, simpleRender(2))

	iv.MoveToBottom()
	if iv.SelectedIndex() != 19 {
		t.Errorf("expected selected 19, got %d", iv.SelectedIndex())
	}

	iv.MoveToTop()
	if iv.SelectedIndex() != 0 {
		t.Errorf("expected selected 0, got %d", iv.SelectedIndex())
	}
}

func TestItemViewport_ShrinkItems(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 10)
	iv.SetItems(10, simpleRender(2))
	iv.MoveDownN(8)

	// Shrink to 5 items — selectedIndex should clamp
	iv.SetItems(5, simpleRender(2))
	if iv.SelectedIndex() != 4 {
		t.Errorf("expected selected 4 after shrink, got %d", iv.SelectedIndex())
	}
}

// TestItemViewport_AnimationAdjustsForContentShift verifies that when content
// height changes during an active spring animation (e.g. image placeholder →
// loaded image), scrollPos is adjusted by the same delta as scrollTarget so
// the animation doesn't jump.
func TestItemViewport_AnimationAdjustsForContentShift(t *testing.T) {
	t.Parallel()

	// 5 items, 3 lines each. Viewport = 6 lines (fits ~2 items).
	iv := NewItemViewport(40, 6)
	iv.SetItems(5, simpleRender(3))

	// Select item 3 and rebuild so the viewport scrolls down.
	iv.MoveDownN(3)
	iv.SetItems(5, simpleRender(3))
	prevOffset := iv.YOffset()

	// Move down to item 4 and start a spring animation.
	iv.MoveDown()
	iv.SetItems(5, simpleRender(3))
	iv.AnimateFrom(prevOffset)

	// Advance the spring a bit so scrollPos is between start and target.
	for range 10 {
		iv.UpdateSpring()
	}
	midPos := iv.YOffset()

	// Now simulate an image loading that adds 2 lines to item 0 (above viewport).
	// This shifts all line offsets below item 0 by +2.
	growItem0 := func(index int, selected bool) string {
		lines := 3
		if index == 0 {
			lines = 5 // grew by 2
		}
		sel := " "
		if selected {
			sel = ">"
		}
		var b strings.Builder
		for j := range lines {
			fmt.Fprintf(&b, "%sitem %d line %d\n", sel, index, j)
		}
		return b.String()
	}
	iv.SetItems(5, growItem0)

	// The viewport YOffset isn't updated until the next spring tick.
	// Advance one frame so the adjusted scrollPos takes effect.
	iv.UpdateSpring()
	newPos := iv.YOffset()
	delta := newPos - midPos

	// The shift should be approximately +2 (the extra lines added to item 0).
	// Allow ±1 for spring rounding and int truncation.
	if delta < 1 || delta > 3 {
		t.Errorf("expected viewport to shift by ~2 after content grew above, got delta=%d (was %d, now %d)", delta, midPos, newPos)
	}
}

func TestItemViewport_ViewContent(t *testing.T) {
	t.Parallel()

	iv := NewItemViewport(40, 6)
	render := func(index int, selected bool) string {
		marker := " "
		if selected {
			marker = ">"
		}
		return fmt.Sprintf("%s[%d]\n", marker, index)
	}
	iv.SetItems(3, render)

	view := iv.View()
	if !strings.Contains(view, ">[0]") {
		t.Errorf("view should show selected marker on item 0, got:\n%s", view)
	}
	if !strings.Contains(view, " [1]") {
		t.Errorf("view should show unselected item 1, got:\n%s", view)
	}
}
