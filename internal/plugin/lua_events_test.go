package plugin

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func setupTestPluginForEvents(perms PermissionSet) (*Plugin, func()) {
	return newTestPluginBase(perms)
}

func TestEventsOnFileOpen(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{EventsFile: true})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		_G.opened_path = ""
		events.on("file.open", function(path)
			_G.opened_path = path
		end)
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(p.EventListeners["file.open"]) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(p.EventListeners["file.open"]))
	}

	p.DispatchEvent("file.open", lua.LString("/tmp/test.go"))

	if p.State.GetGlobal("opened_path").String() != "/tmp/test.go" {
		t.Errorf("expected '/tmp/test.go', got %q", p.State.GetGlobal("opened_path").String())
	}
}

func TestEventsOnEditorChange(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{EventsEditor: true})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		_G.change_count = 0
		events.on("editor.change", function()
			_G.change_count = _G.change_count + 1
		end)
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	p.DispatchEvent("editor.change")
	p.DispatchEvent("editor.change")

	if p.State.GetGlobal("change_count").String() != "2" {
		t.Errorf("expected 2 changes, got %s", p.State.GetGlobal("change_count").String())
	}
}

func TestEventsFileWithoutPermission(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		events.on("file.open", function() end)
	`)
	if err == nil {
		t.Fatal("expected error when events.file not granted")
	}
}

func TestEventsEditorWithoutPermission(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{EventsFile: true})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		events.on("editor.change", function() end)
	`)
	if err == nil {
		t.Fatal("expected error when events.editor not granted")
	}
}

func TestEventsUnknownEvent(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{EventsFile: true, EventsEditor: true})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		events.on("unknown.event", function() end)
	`)
	if err == nil {
		t.Fatal("expected error for unknown event")
	}
}

func TestEventsMultipleListeners(t *testing.T) {
	p, cleanup := setupTestPluginForEvents(PermissionSet{EventsFile: true})
	defer cleanup()

	err := p.State.DoString(`
		local events = require("ttt.events")
		_G.log = ""
		events.on("file.save", function(path)
			_G.log = _G.log .. "A:" .. path .. ";"
		end)
		events.on("file.save", function(path)
			_G.log = _G.log .. "B:" .. path .. ";"
		end)
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	p.DispatchEvent("file.save", lua.LString("test.go"))

	expected := "A:test.go;B:test.go;"
	if p.State.GetGlobal("log").String() != expected {
		t.Errorf("expected %q, got %q", expected, p.State.GetGlobal("log").String())
	}
}
