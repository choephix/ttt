package plugin

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	lua "github.com/yuin/gopher-lua"
)

func TestEventToLuaKeyEvent(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	tbl := eventToLua(L, ev)
	if tbl == nil {
		t.Fatal("expected non-nil table")
	}

	if tbl.RawGetString("type").String() != "key" {
		t.Error("expected type=key")
	}
	if tbl.RawGetString("key").String() != "a" {
		t.Errorf("expected key=a, got %s", tbl.RawGetString("key").String())
	}
}

func TestEventToLuaKeyEventSpecial(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	tbl := eventToLua(L, ev)
	if tbl == nil {
		t.Fatal("expected non-nil table")
	}

	key := tbl.RawGetString("key").String()
	if key != "Enter" {
		t.Errorf("expected key=Enter, got %s", key)
	}
}

func TestEventToLuaKeyEventModNilWhenNone(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	tbl := eventToLua(L, ev)

	mod := tbl.RawGetString("mod")
	if mod != lua.LNil {
		t.Errorf("expected mod to be nil when no modifier, got %s (%q)", mod.Type(), mod.String())
	}
}

func TestEventToLuaKeyEventCtrl(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModCtrl)
	tbl := eventToLua(L, ev)

	mod := tbl.RawGetString("mod").String()
	if mod != "ctrl" {
		t.Errorf("expected mod=ctrl, got %s", mod)
	}
}

func TestEventToLuaKeyEventCombinedModifiers(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModCtrl|tcell.ModShift)
	tbl := eventToLua(L, ev)

	mod := tbl.RawGetString("mod").String()
	if mod != "ctrl+shift" {
		t.Errorf("expected mod=ctrl+shift, got %s", mod)
	}
}

func TestEventToLuaMouseEvent(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ev := tcell.NewEventMouse(5, 10, tcell.Button1, tcell.ModNone)
	tbl := eventToLua(L, ev)
	if tbl == nil {
		t.Fatal("expected non-nil table")
	}

	if tbl.RawGetString("type").String() != "mouse" {
		t.Error("expected type=mouse")
	}
	if int(tbl.RawGetString("x").(lua.LNumber)) != 5 {
		t.Error("expected x=5")
	}
	if int(tbl.RawGetString("y").(lua.LNumber)) != 10 {
		t.Error("expected y=10")
	}
	if tbl.RawGetString("button").String() != "left" {
		t.Error("expected button=left")
	}
}
