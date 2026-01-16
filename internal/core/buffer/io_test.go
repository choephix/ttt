package buffer

import (
	"os"
	"testing"
)

func TestLoadAndSaveFile(t *testing.T) {
	fname := "testfile.txt"
	defer os.Remove(fname)
	orig := &Buffer{Lines: []string{"hello", "world"}}
	if err := orig.SaveFile(fname); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	b := &Buffer{}
	if err := b.LoadFile(fname); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if len(b.Lines) != 2 || b.Lines[0] != "hello" || b.Lines[1] != "world" {
		t.Errorf("expected [hello world], got %v", b.Lines)
	}
}
