package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginFilesystemAPI_PathRestriction(t *testing.T) {
	tmpDir := t.TempDir()
	allowed := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(allowed, 0755)
	os.WriteFile(filepath.Join(allowed, "test.txt"), []byte("hello"), 0644)

	forbidden := filepath.Join(tmpDir, "secrets")
	os.MkdirAll(forbidden, 0755)
	os.WriteFile(filepath.Join(forbidden, "key.pem"), []byte("secret"), 0644)

	api := NewPluginFilesystemAPI(allowed)

	t.Run("read allowed", func(t *testing.T) {
		content, err := api.ReadFile(filepath.Join(allowed, "test.txt"))
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
		if content != "hello" {
			t.Errorf("expected 'hello', got %q", content)
		}
	})

	t.Run("read forbidden", func(t *testing.T) {
		_, err := api.ReadFile(filepath.Join(forbidden, "key.pem"))
		if err == nil {
			t.Fatal("expected access denied error")
		}
	})

	t.Run("write allowed", func(t *testing.T) {
		err := api.WriteFile(filepath.Join(allowed, "out.txt"), "data")
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
	})

	t.Run("write forbidden", func(t *testing.T) {
		err := api.WriteFile(filepath.Join(forbidden, "evil.txt"), "data")
		if err == nil {
			t.Fatal("expected access denied error")
		}
	})

	t.Run("exists allowed", func(t *testing.T) {
		if !api.FileExists(filepath.Join(allowed, "test.txt")) {
			t.Error("expected true for existing allowed file")
		}
	})

	t.Run("exists forbidden returns false", func(t *testing.T) {
		if api.FileExists(filepath.Join(forbidden, "key.pem")) {
			t.Error("expected false for forbidden path")
		}
	})

	t.Run("list allowed", func(t *testing.T) {
		entries, err := api.ListDir(allowed)
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
		if len(entries) == 0 {
			t.Error("expected entries in allowed dir")
		}
	})

	t.Run("list forbidden", func(t *testing.T) {
		_, err := api.ListDir(forbidden)
		if err == nil {
			t.Fatal("expected access denied error")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		_, err := api.ReadFile(filepath.Join(allowed, "..", "secrets", "key.pem"))
		if err == nil {
			t.Fatal("expected path traversal to be blocked")
		}
	})

	t.Run("absolute path outside roots blocked", func(t *testing.T) {
		_, err := api.ReadFile("/etc/passwd")
		if err == nil {
			t.Fatal("expected absolute path outside roots to be blocked")
		}
	})
}

func TestPluginFilesystemAPI_MultipleRoots(t *testing.T) {
	tmpDir := t.TempDir()
	root1 := filepath.Join(tmpDir, "project")
	root2 := filepath.Join(tmpDir, "plugin")
	outside := filepath.Join(tmpDir, "outside")

	for _, d := range []string{root1, root2, outside} {
		os.MkdirAll(d, 0755)
		os.WriteFile(filepath.Join(d, "file.txt"), []byte("data"), 0644)
	}

	api := NewPluginFilesystemAPI(root1, root2)

	if _, err := api.ReadFile(filepath.Join(root1, "file.txt")); err != nil {
		t.Errorf("root1 should be accessible: %v", err)
	}
	if _, err := api.ReadFile(filepath.Join(root2, "file.txt")); err != nil {
		t.Errorf("root2 should be accessible: %v", err)
	}
	if _, err := api.ReadFile(filepath.Join(outside, "file.txt")); err == nil {
		t.Error("outside should not be accessible")
	}
}

func TestPluginFilesystemAPI_SymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()
	allowed := filepath.Join(tmpDir, "workspace")
	secrets := filepath.Join(tmpDir, "secrets")
	os.MkdirAll(allowed, 0755)
	os.MkdirAll(secrets, 0755)
	os.WriteFile(filepath.Join(secrets, "key.pem"), []byte("secret"), 0644)

	symlink := filepath.Join(allowed, "escape")
	if err := os.Symlink(secrets, symlink); err != nil {
		t.Skip("symlinks not supported")
	}

	api := NewPluginFilesystemAPI(allowed)

	_, err := api.ReadFile(filepath.Join(symlink, "key.pem"))
	if err == nil {
		t.Fatal("expected symlink escape to be blocked")
	}
}
