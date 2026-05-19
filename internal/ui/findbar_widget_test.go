package ui

import "testing"

func TestFindInLines(t *testing.T) {
	lines := []string{"hello world", "foo bar", "hello again"}
	matches := FindInLines(lines, "hello")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Line != 0 || matches[0].Col != 0 {
		t.Fatalf("match 0 wrong: %+v", matches[0])
	}
	if matches[1].Line != 2 || matches[1].Col != 0 {
		t.Fatalf("match 1 wrong: %+v", matches[1])
	}
}

func TestFindInLinesCaseInsensitive(t *testing.T) {
	lines := []string{"Hello World", "HELLO"}
	matches := FindInLines(lines, "hello")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestFindInLinesMultiplePerLine(t *testing.T) {
	lines := []string{"abcabc"}
	matches := FindInLines(lines, "abc")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Col != 0 || matches[1].Col != 3 {
		t.Fatalf("unexpected cols: %d, %d", matches[0].Col, matches[1].Col)
	}
}

func TestFindInLinesEmpty(t *testing.T) {
	lines := []string{"hello"}
	matches := FindInLines(lines, "")
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for empty query, got %d", len(matches))
	}
}

func TestFindInLinesNoMatch(t *testing.T) {
	lines := []string{"hello world"}
	matches := FindInLines(lines, "xyz")
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}
