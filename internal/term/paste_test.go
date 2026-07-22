package term

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func key(k tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(k, r, mod)
}

func runeKey(r rune) *tcell.EventKey {
	return key(tcell.KeyRune, r, tcell.ModNone)
}

// keysFromString simulates how tcell encodes each byte of a string during
// bracketed paste. This mirrors tcell's input.go inpStateInit switch exactly:
//   - \t → KeyTab
//   - \r → KeyEnter
//   - \b, 0x7F → KeyBackspace
//   - r < ' ' → KeyCtrlSpace+Key(r) with ModCtrl  (covers \n as KeyCtrlJ)
//   - r >= ' ' → KeyRune
func keysFromString(s string) []*tcell.EventKey {
	var events []*tcell.EventKey
	for _, r := range s {
		switch {
		case r == '\t':
			events = append(events, key(tcell.KeyTab, 0, tcell.ModNone))
		case r == '\b' || r == 0x7F:
			events = append(events, key(tcell.KeyBackspace, 0, tcell.ModNone))
		case r == '\r':
			events = append(events, key(tcell.KeyEnter, 0, tcell.ModNone))
		case r < ' ':
			events = append(events, key(tcell.KeyCtrlSpace+tcell.Key(r), 0, tcell.ModCtrl))
		default:
			events = append(events, runeKey(r))
		}
	}
	return events
}

// --- Basic functionality ---

func TestCollectPasteText_Runes(t *testing.T) {
	got := CollectPasteText(keysFromString("Hi"))
	if got != "Hi" {
		t.Errorf("got %q, want %q", got, "Hi")
	}
}

func TestCollectPasteText_Empty(t *testing.T) {
	got := CollectPasteText(nil)
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestCollectPasteText_Tab(t *testing.T) {
	got := CollectPasteText(keysFromString("\tx"))
	if got != "\tx" {
		t.Errorf("got %q, want %q", got, "\tx")
	}
}

// --- Full ASCII printable range ---

func TestCollectPasteText_AllPrintableASCII(t *testing.T) {
	var want strings.Builder
	for r := rune(' '); r <= '~'; r++ {
		want.WriteRune(r)
	}
	input := want.String()
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("printable ASCII mismatch:\ngot  %q\nwant %q", got, input)
	}
}

func TestCollectPasteText_Digits(t *testing.T) {
	got := CollectPasteText(keysFromString("0123456789"))
	if got != "0123456789" {
		t.Errorf("got %q, want %q", got, "0123456789")
	}
}

func TestCollectPasteText_Punctuation(t *testing.T) {
	input := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

// --- Line ending normalization ---

func TestCollectPasteText_UnixNewlines(t *testing.T) {
	got := CollectPasteText(keysFromString("a\nb"))
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_CRLFNewlines(t *testing.T) {
	got := CollectPasteText(keysFromString("a\r\nb"))
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_CROnly(t *testing.T) {
	got := CollectPasteText(keysFromString("a\rb"))
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_MixedLineEndings(t *testing.T) {
	got := CollectPasteText(keysFromString("a\r\nb\nc\rd"))
	if got != "a\nb\nc\nd" {
		t.Errorf("got %q, want %q", got, "a\nb\nc\nd")
	}
}

func TestCollectPasteText_ConsecutiveLF(t *testing.T) {
	got := CollectPasteText(keysFromString("a\n\nb"))
	if got != "a\n\nb" {
		t.Errorf("got %q, want %q", got, "a\n\nb")
	}
}

func TestCollectPasteText_ConsecutiveCRLF(t *testing.T) {
	got := CollectPasteText(keysFromString("a\r\n\r\nb"))
	if got != "a\n\nb" {
		t.Errorf("got %q, want %q", got, "a\n\nb")
	}
}

func TestCollectPasteText_ConsecutiveCR(t *testing.T) {
	got := CollectPasteText(keysFromString("a\r\rb"))
	if got != "a\n\nb" {
		t.Errorf("got %q, want %q", got, "a\n\nb")
	}
}

func TestCollectPasteText_OnlyNewlines(t *testing.T) {
	got := CollectPasteText(keysFromString("\n\r\n\r"))
	if got != "\n\n\n" {
		t.Errorf("got %q, want %q", got, "\n\n\n")
	}
}

func TestCollectPasteText_TrailingNewline(t *testing.T) {
	got := CollectPasteText(keysFromString("hello\n"))
	if got != "hello\n" {
		t.Errorf("got %q, want %q", got, "hello\n")
	}
}

func TestCollectPasteText_TrailingCRLF(t *testing.T) {
	got := CollectPasteText(keysFromString("hello\r\n"))
	if got != "hello\n" {
		t.Errorf("got %q, want %q", got, "hello\n")
	}
}

// --- Unicode coverage ---

func TestCollectPasteText_BoxDrawing(t *testing.T) {
	input := "┌──────┐\n│ test │\n└──────┘"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_CJK(t *testing.T) {
	input := "你好世界"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_Emoji(t *testing.T) {
	input := "Hello 🌍🎉🚀"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_EmojiSequences(t *testing.T) {
	input := "👨‍👩‍👧‍👦 family"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_CombiningCharacters(t *testing.T) {
	input := "é ñ" // é ñ via combining marks
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_Arabic(t *testing.T) {
	input := "مرحبا بالعالم"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_Japanese(t *testing.T) {
	input := "こんにちは世界"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_Korean(t *testing.T) {
	input := "안녕하세요"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_MathSymbols(t *testing.T) {
	input := "∑∏∫∂√∞≈≠≤≥"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_CurrencySymbols(t *testing.T) {
	input := "$€£¥₹₿"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_MixedScripts(t *testing.T) {
	input := "Hello 你好 مرحبا こんにちは 🌍"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

// --- Control character handling (intentionally dropped) ---
// tcell encodes control chars (0x00-0x1F except \t,\r,\n) as KeyCtrl*
// and we intentionally skip them. These tests lock down that behavior.

func TestCollectPasteText_NullByteDropped(t *testing.T) {
	events := []*tcell.EventKey{
		runeKey('a'),
		key(tcell.KeyCtrlSpace, 0, tcell.ModCtrl), // 0x00
		runeKey('b'),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestCollectPasteText_AllControlCharsDropped(t *testing.T) {
	// 0x01 through 0x08 (CtrlA-CtrlH), 0x0B-0x0C (CtrlK-CtrlL),
	// 0x0E-0x1A (CtrlN-CtrlZ) — all should be dropped
	droppedKeys := []tcell.Key{
		tcell.KeyCtrlA, tcell.KeyCtrlB, tcell.KeyCtrlC, tcell.KeyCtrlD,
		tcell.KeyCtrlE, tcell.KeyCtrlF, tcell.KeyCtrlG, tcell.KeyCtrlH,
		tcell.KeyCtrlK, tcell.KeyCtrlL,
		tcell.KeyCtrlN, tcell.KeyCtrlO, tcell.KeyCtrlP, tcell.KeyCtrlQ,
		tcell.KeyCtrlR, tcell.KeyCtrlS, tcell.KeyCtrlT, tcell.KeyCtrlU,
		tcell.KeyCtrlV, tcell.KeyCtrlW, tcell.KeyCtrlX, tcell.KeyCtrlY,
		tcell.KeyCtrlZ,
	}
	for _, dk := range droppedKeys {
		events := []*tcell.EventKey{
			runeKey('x'),
			key(dk, 0, tcell.ModCtrl),
			runeKey('y'),
		}
		got := CollectPasteText(events)
		if got != "xy" {
			t.Errorf("key %d not dropped: got %q, want %q", dk, got, "xy")
		}
	}
}

func TestCollectPasteText_BackspaceDropped(t *testing.T) {
	events := []*tcell.EventKey{
		runeKey('a'),
		key(tcell.KeyBackspace, 0, tcell.ModNone), // 0x7F
		runeKey('b'),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestCollectPasteText_EscapeDropped(t *testing.T) {
	events := []*tcell.EventKey{
		runeKey('a'),
		key(tcell.KeyEscape, 0, tcell.ModNone),
		runeKey('b'),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestCollectPasteText_ArrowKeysDropped(t *testing.T) {
	events := []*tcell.EventKey{
		runeKey('a'),
		key(tcell.KeyUp, 0, tcell.ModNone),
		key(tcell.KeyDown, 0, tcell.ModNone),
		key(tcell.KeyLeft, 0, tcell.ModNone),
		key(tcell.KeyRight, 0, tcell.ModNone),
		runeKey('b'),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestCollectPasteText_FunctionKeysDropped(t *testing.T) {
	events := []*tcell.EventKey{
		runeKey('a'),
		key(tcell.KeyF1, 0, tcell.ModNone),
		key(tcell.KeyF12, 0, tcell.ModNone),
		key(tcell.KeyHome, 0, tcell.ModNone),
		key(tcell.KeyEnd, 0, tcell.ModNone),
		key(tcell.KeyPgUp, 0, tcell.ModNone),
		key(tcell.KeyPgDn, 0, tcell.ModNone),
		key(tcell.KeyInsert, 0, tcell.ModNone),
		key(tcell.KeyDelete, 0, tcell.ModNone),
		runeKey('b'),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

// --- Realistic paste scenarios ---

func TestCollectPasteText_GoCodeSnippet(t *testing.T) {
	input := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_PythonCodeSnippet(t *testing.T) {
	input := "def greet(name):\n    print(f\"Hello, {name}!\")\n\ngreet(\"world\")\n"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_HTMLSnippet(t *testing.T) {
	input := "<div class=\"container\">\n\t<p>Hello &amp; world</p>\n</div>"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_JSONSnippet(t *testing.T) {
	input := "{\n\t\"name\": \"ttt\",\n\t\"version\": \"1.0.0\",\n\t\"tags\": [\"editor\", \"terminal\"]\n}"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_WindowsCodeSnippet(t *testing.T) {
	input := "func main() {\r\n\tfmt.Println(\"hello\")\r\n}\r\n"
	want := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := CollectPasteText(keysFromString(input))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCollectPasteText_ShellCommand(t *testing.T) {
	input := "grep -rn 'TODO' . | sort | head -20"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_URLWithSpecialChars(t *testing.T) {
	input := "https://example.com/search?q=hello+world&lang=en#results"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_FilePath(t *testing.T) {
	input := "/home/user/.config/ttt/settings.json"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_WindowsPath(t *testing.T) {
	input := "C:\\Users\\dev\\Documents\\project\\main.go"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

// --- Scale tests ---

func TestCollectPasteText_LargePaste(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 1000; i++ {
		input.WriteString("Line with unicode: 你好 émoji 🎉 and tabs\t!\n")
	}
	want := input.String()
	got := CollectPasteText(keysFromString(want))
	if got != want {
		t.Errorf("large paste: length got %d, want %d", len(got), len(want))
	}
}

func TestCollectPasteText_SingleCharacter(t *testing.T) {
	got := CollectPasteText(keysFromString("x"))
	if got != "x" {
		t.Errorf("got %q, want %q", got, "x")
	}
}

func TestCollectPasteText_SingleNewline(t *testing.T) {
	got := CollectPasteText(keysFromString("\n"))
	if got != "\n" {
		t.Errorf("got %q, want %q", got, "\n")
	}
}

func TestCollectPasteText_SingleTab(t *testing.T) {
	got := CollectPasteText(keysFromString("\t"))
	if got != "\t" {
		t.Errorf("got %q, want %q", got, "\t")
	}
}

// --- Whitespace handling ---

func TestCollectPasteText_MixedWhitespace(t *testing.T) {
	input := "  \t  \t\t  "
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_IndentedCode(t *testing.T) {
	input := "\t\tif x > 0 {\n\t\t\treturn x\n\t\t}\n"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestCollectPasteText_SpaceIndentedCode(t *testing.T) {
	input := "    def foo():\n        return 42\n"
	got := CollectPasteText(keysFromString(input))
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}
