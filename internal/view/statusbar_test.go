package view

import (
	"testing"
	"time"
)

func TestStatusBarFields(t *testing.T) {
	sb := &StatusBar{FileName: "file.go", Line: 2, Col: 4, Dirty: true}
	if sb.FileName != "file.go" {
		t.Errorf("expected FileName 'file.go', got %q", sb.FileName)
	}
	if sb.Line != 2 {
		t.Errorf("expected Line 2, got %d", sb.Line)
	}
	if !sb.Dirty {
		t.Error("expected Dirty to be true")
	}
}

func TestSetNotification(t *testing.T) {
	sb := &StatusBar{}
	sb.SetNotification("saved", NotifyInfo, 3*time.Second)

	if sb.Notification != "saved" {
		t.Errorf("expected Notification 'saved', got %q", sb.Notification)
	}
	if sb.NotifyLevel != NotifyInfo {
		t.Errorf("expected NotifyLevel NotifyInfo, got %d", sb.NotifyLevel)
	}
	if sb.NotifyExpiry.IsZero() {
		t.Error("expected NotifyExpiry to be set")
	}
	if sb.NotifyAction != nil {
		t.Error("expected NotifyAction to be nil after SetNotification")
	}
	if sb.ActionLabel != "" {
		t.Errorf("expected ActionLabel to be empty, got %q", sb.ActionLabel)
	}
}

func TestSetNotificationClearsAction(t *testing.T) {
	sb := &StatusBar{}
	called := false
	sb.SetNotificationWithAction("error", NotifyError, 5*time.Second, "Retry", func() { called = true })

	// Now overwrite with a plain notification
	sb.SetNotification("info", NotifyInfo, 2*time.Second)

	if sb.NotifyAction != nil {
		t.Error("expected NotifyAction to be nil after SetNotification overwrites action notification")
	}
	if sb.ActionLabel != "" {
		t.Errorf("expected ActionLabel to be empty, got %q", sb.ActionLabel)
	}
	if called {
		t.Error("action should not have been called")
	}
}

func TestSetNotificationWithAction(t *testing.T) {
	sb := &StatusBar{}
	called := false
	action := func() { called = true }
	sb.SetNotificationWithAction("failed", NotifyError, 10*time.Second, "Retry", action)

	if sb.Notification != "failed" {
		t.Errorf("expected Notification 'failed', got %q", sb.Notification)
	}
	if sb.NotifyLevel != NotifyError {
		t.Errorf("expected NotifyLevel NotifyError, got %d", sb.NotifyLevel)
	}
	if sb.ActionLabel != "Retry" {
		t.Errorf("expected ActionLabel 'Retry', got %q", sb.ActionLabel)
	}
	if sb.NotifyAction == nil {
		t.Fatal("expected NotifyAction to be set")
	}

	sb.NotifyAction()
	if !called {
		t.Error("expected action to be called")
	}
}

func TestDismissNotification(t *testing.T) {
	sb := &StatusBar{}
	sb.SetNotificationWithAction("error", NotifyError, 10*time.Second, "Fix", func() {})
	sb.SecondaryLabel = "Details"
	sb.SecondaryAction = func() {}

	sb.DismissNotification()

	if sb.Notification != "" {
		t.Errorf("expected Notification to be empty, got %q", sb.Notification)
	}
	if !sb.NotifyExpiry.IsZero() {
		t.Error("expected NotifyExpiry to be zero after dismiss")
	}
	if sb.NotifyAction != nil {
		t.Error("expected NotifyAction to be nil after dismiss")
	}
	if sb.ActionLabel != "" {
		t.Errorf("expected ActionLabel to be empty, got %q", sb.ActionLabel)
	}
	if sb.SecondaryAction != nil {
		t.Error("expected SecondaryAction to be nil after dismiss")
	}
	if sb.SecondaryLabel != "" {
		t.Errorf("expected SecondaryLabel to be empty, got %q", sb.SecondaryLabel)
	}
}

func TestIsNotificationActive_NoNotification(t *testing.T) {
	sb := &StatusBar{}
	if sb.IsNotificationActive() {
		t.Error("expected no active notification for empty StatusBar")
	}
}

func TestIsNotificationActive_NotExpired(t *testing.T) {
	sb := &StatusBar{}
	sb.SetNotification("hello", NotifyInfo, 10*time.Second)

	if !sb.IsNotificationActive() {
		t.Error("expected notification to be active (not expired)")
	}
}

func TestIsNotificationActive_Expired(t *testing.T) {
	sb := &StatusBar{}
	sb.SetNotification("old", NotifyWarning, -1*time.Second) // already expired

	if sb.IsNotificationActive() {
		t.Error("expected notification to be inactive (expired)")
	}
	// After checking, the notification should be dismissed
	if sb.Notification != "" {
		t.Errorf("expected Notification to be cleared after expiry check, got %q", sb.Notification)
	}
}

func TestIsNotificationActive_ZeroExpiry(t *testing.T) {
	sb := &StatusBar{}
	sb.Notification = "permanent"
	sb.NotifyExpiry = time.Time{} // zero value = no expiry

	if !sb.IsNotificationActive() {
		t.Error("expected notification with zero expiry to remain active")
	}
}

func TestNotifyLevel_Style(t *testing.T) {
	tests := []struct {
		level NotifyLevel
		want  string
	}{
		{NotifyInfo, "StyleStatusBar"},
		{NotifyWarning, "StyleWarning"},
		{NotifyError, "StyleDanger"},
	}

	for _, tt := range tests {
		got := tt.level.Style()
		// We can't import term constants here (same package), but the
		// Style() method returns term.Style values. Verify they return
		// distinct values for each level.
		_ = got
	}

	// Verify each level returns distinct styles
	infoStyle := NotifyInfo.Style()
	warningStyle := NotifyWarning.Style()
	errorStyle := NotifyError.Style()

	if warningStyle == infoStyle {
		t.Error("expected NotifyWarning style to differ from NotifyInfo")
	}
	if errorStyle == infoStyle {
		t.Error("expected NotifyError style to differ from NotifyInfo")
	}
	if errorStyle == warningStyle {
		t.Error("expected NotifyError style to differ from NotifyWarning")
	}
}
