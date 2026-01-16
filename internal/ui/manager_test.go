package ui

import "testing"

func TestWindowManager_AddAndSwitch(t *testing.T) {
	m := &WindowManager{}
	w1 := &Window{}
	w2 := &Window{}
	m.AddWindow(w1)
	m.AddWindow(w2)
	if m.Focus != 1 {
		t.Errorf("expected focus=1, got %d", m.Focus)
	}
	m.SwitchFocus(0)
	if m.Current() != w1 {
		t.Error("expected current window to be w1")
	}
}
