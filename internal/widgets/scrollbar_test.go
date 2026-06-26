package widgets

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

func TestScrollbarThumbAtTop(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}
	top, _ := sb.thumbPos()
	if top != 0 {
		t.Fatalf("expected thumb top=0 when scrolled to top, got %d", top)
	}
}

func TestScrollbarThumbAtBottom(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    90, // max scroll = TotalItems - Height
	}
	top, thumbH := sb.thumbPos()
	if top+thumbH != sb.Height {
		t.Fatalf("expected thumb bottom at %d, got top=%d thumbH=%d (bottom=%d)", sb.Height, top, thumbH, top+thumbH)
	}
}

func TestScrollbarThumbPositionMidScroll(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    45, // roughly half
	}
	top, thumbH := sb.thumbPos()
	// Thumb should be in the middle area, not at 0 and not at bottom
	if top <= 0 {
		t.Errorf("expected thumb in middle, got top=%d", top)
	}
	if top+thumbH >= sb.Height {
		t.Errorf("expected thumb not at bottom, got top=%d thumbH=%d", top, thumbH)
	}
}

func TestScrollbarThumbHeightMinimum(t *testing.T) {
	sb := &scrollbar{
		Height:     5,
		TotalItems: 10000,
		TopItem:    0,
	}
	_, thumbH := sb.thumbPos()
	if thumbH < 1 {
		t.Fatalf("thumb height should be at least 1, got %d", thumbH)
	}
}

func TestScrollbarNotVisibleWhenContentFits(t *testing.T) {
	sb := &scrollbar{
		Height:     20,
		TotalItems: 10, // fewer items than height
		TopItem:    0,
	}
	if sb.visible() {
		t.Fatal("scrollbar should not be visible when content fits in view")
	}
}

func TestScrollbarVisibleWhenContentOverflows(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}
	if !sb.visible() {
		t.Fatal("scrollbar should be visible when content overflows")
	}
}

func TestScrollbarRender(t *testing.T) {
	sb := &scrollbar{
		X:          0,
		Y:          0,
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}
	s := newTestSurface(1, 10)
	sb.Render(s, 0, 0)

	thumbTop, thumbH := sb.thumbPos()

	// Check that thumb cells use StyleScrollbarThumb and track cells use StyleScrollbar
	for y := 0; y < 10; y++ {
		expected := term.StyleScrollbar
		if y >= thumbTop && y < thumbTop+thumbH {
			expected = term.StyleScrollbarThumb
		}
		if s.cells[y][0].Style != expected {
			t.Errorf("row %d: expected style %d, got %d", y, expected, s.cells[y][0].Style)
		}
	}
}

func TestScrollbarRenderNotVisibleNoOp(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 5, // fits
		TopItem:    0,
	}
	s := newTestSurface(1, 10)
	sb.Render(s, 0, 0)

	// All cells should remain zero-value (not rendered)
	for y := 0; y < 10; y++ {
		if s.cells[y][0].Ch != 0 {
			t.Errorf("row %d should be empty when scrollbar is not visible", y)
		}
	}
}

func TestScrollbarClickOnThumb(t *testing.T) {
	sb := &scrollbar{
		X:          5,
		Y:          0,
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	// Click on the scrollbar column at its top position
	click := tcell.NewEventMouse(5, 0, tcell.Button1, tcell.ModNone)
	_, consumed := sb.HandleEvent(click)
	if !consumed {
		t.Fatal("click on scrollbar should be consumed")
	}
	if !sb.dragging {
		t.Fatal("scrollbar should enter dragging state after click")
	}
}

func TestScrollbarClickOutside(t *testing.T) {
	sb := &scrollbar{
		X:          5,
		Y:          0,
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	// Click away from scrollbar column
	click := tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone)
	_, consumed := sb.HandleEvent(click)
	if consumed {
		t.Fatal("click outside scrollbar should not be consumed")
	}
}

func TestScrollbarDragUpdatesTopItem(t *testing.T) {
	sb := &scrollbar{
		X:          0,
		Y:          0,
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	// Click on scrollbar to start drag
	click := tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone)
	sb.HandleEvent(click)

	// Drag to the bottom
	drag := tcell.NewEventMouse(0, 9, tcell.Button1, tcell.ModNone)
	newTop, consumed := sb.HandleEvent(drag)
	if !consumed {
		t.Fatal("drag should be consumed")
	}
	if newTop <= 0 {
		t.Fatalf("dragging to bottom should increase topItem, got %d", newTop)
	}

	// Release
	release := tcell.NewEventMouse(0, 9, tcell.ButtonNone, tcell.ModNone)
	_, consumed = sb.HandleEvent(release)
	if consumed {
		t.Fatal("release should not be consumed (ends drag)")
	}
	if sb.dragging {
		t.Fatal("scrollbar should exit dragging state on release")
	}
}

func TestScrollbarClickBelowThumb(t *testing.T) {
	sb := &scrollbar{
		X:          0,
		Y:          0,
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	// Thumb starts at top. Click below thumb area to jump.
	_, thumbH := sb.thumbPos()
	clickY := thumbH + 2 // below the thumb
	click := tcell.NewEventMouse(0, clickY, tcell.Button1, tcell.ModNone)
	newTop, consumed := sb.HandleEvent(click)
	if !consumed {
		t.Fatal("click on scrollbar track should be consumed")
	}
	if newTop <= 0 {
		t.Fatalf("clicking below thumb should scroll down, got topItem=%d", newTop)
	}
}

func TestScrollbarKeyEventIgnored(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	_, consumed := sb.HandleEvent(ev)
	if consumed {
		t.Fatal("key events should not be consumed by scrollbar")
	}
}

func TestScrollbarPosToTopItemClamping(t *testing.T) {
	sb := &scrollbar{
		Height:     10,
		TotalItems: 100,
		TopItem:    0,
	}

	// Negative thumbTop should clamp to 0
	top := sb.posToTopItem(-10)
	if top != 0 {
		t.Fatalf("expected clamped topItem=0 for negative pos, got %d", top)
	}

	// Very large thumbTop should clamp to max scrollable
	top = sb.posToTopItem(1000)
	maxScrollable := sb.TotalItems - sb.Height
	if top != maxScrollable {
		t.Fatalf("expected clamped topItem=%d for huge pos, got %d", maxScrollable, top)
	}
}
