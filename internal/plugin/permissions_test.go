package plugin

import "testing"

func TestDiffPermissionsEmpty(t *testing.T) {
	a := PermissionSet{PanelSidebar: true, Commands: true}
	b := PermissionSet{PanelSidebar: true, Commands: true}

	diff := DiffPermissions(a, b)
	if !diff.IsEmpty() {
		t.Errorf("expected empty diff, got %d entries", len(diff.Entries))
	}
}

func TestDiffPermissionsNewBool(t *testing.T) {
	granted := PermissionSet{PanelSidebar: true}
	requested := PermissionSet{PanelSidebar: true, Commands: true, FsRead: true}

	diff := DiffPermissions(granted, requested)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	if len(diff.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(diff.Entries))
	}

	names := map[string]bool{}
	for _, e := range diff.Entries {
		names[e.Name] = true
	}
	if !names["commands"] {
		t.Error("missing diff entry for commands")
	}
	if !names["fs.read"] {
		t.Error("missing diff entry for fs.read")
	}
}

func TestDiffPermissionsNewExec(t *testing.T) {
	granted := PermissionSet{SystemExec: []string{"docker"}}
	requested := PermissionSet{SystemExec: []string{"docker", "kubectl"}}

	diff := DiffPermissions(granted, requested)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	if len(diff.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(diff.Entries))
	}
	if diff.Entries[0].Value != "kubectl" {
		t.Errorf("expected kubectl, got %s", diff.Entries[0].Value)
	}
}

func TestDiffPermissionsSubset(t *testing.T) {
	granted := PermissionSet{PanelSidebar: true, Commands: true, FsRead: true}
	requested := PermissionSet{PanelSidebar: true}

	diff := DiffPermissions(granted, requested)
	if !diff.IsEmpty() {
		t.Errorf("expected empty diff for subset, got %d entries", len(diff.Entries))
	}
}

func TestCheckPermission(t *testing.T) {
	ps := PermissionSet{PanelSidebar: true}

	if err := ps.Check("panel.sidebar"); err != nil {
		t.Errorf("expected allowed, got %v", err)
	}
	if err := ps.Check("commands"); err == nil {
		t.Error("expected denied for commands")
	}
}

func TestCheckExec(t *testing.T) {
	ps := PermissionSet{SystemExec: []string{"docker", "git"}}

	if err := ps.CheckExec("docker"); err != nil {
		t.Errorf("expected allowed, got %v", err)
	}
	if err := ps.CheckExec("rm"); err == nil {
		t.Error("expected denied for rm")
	}
}

func TestDisplayEntries(t *testing.T) {
	ps := PermissionSet{
		PanelSidebar: true,
		Commands:     true,
		SystemExec:   []string{"docker"},
	}

	entries := ps.DisplayEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 display entries, got %d", len(entries))
	}
}
