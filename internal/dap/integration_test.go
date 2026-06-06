package dap

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func findDlv() string {
	if path, err := exec.LookPath("dlv"); err == nil {
		return path
	}
	home, _ := os.UserHomeDir()
	candidate := filepath.Join(home, "go", "bin", "dlv")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func TestIntegrationDelve(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dlv := findDlv()
	if dlv == "" {
		t.Skip("dlv not found, install with: go install github.com/go-delve/delve/cmd/dlv@latest")
	}

	testdata, _ := filepath.Abs("testdata/hello.go")
	if _, err := os.Stat(testdata); err != nil {
		t.Fatalf("test program not found: %s", testdata)
	}
	workDir := filepath.Dir(testdata)

	client, err := NewTCPClient([]string{dlv, "dap"}, workDir)
	if err != nil {
		t.Fatalf("start dlv: %v", err)
	}
	defer client.Close()

	// Initialize
	if err := client.Initialize("go"); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	caps := client.Capabilities()
	if !caps.SupportsConfigurationDoneRequest {
		t.Error("expected dlv to support configurationDone")
	}

	// Set up event handlers
	stopped := make(chan StoppedEventBody, 1)
	client.OnStopped = func(body StoppedEventBody) {
		stopped <- body
	}
	exited := make(chan ExitedEventBody, 1)
	client.OnExited = func(body ExitedEventBody) {
		exited <- body
	}
	terminated := make(chan struct{}, 1)
	client.OnTerminated = func(body TerminatedEventBody) {
		terminated <- struct{}{}
	}

	// Launch first — dlv expects launch before breakpoints
	if err := client.Launch(testdata, false, nil); err != nil {
		t.Fatalf("launch: %v", err)
	}

	// Set breakpoint on line 7 (y := "hello")
	bps, err := client.SetBreakpoints(testdata, []SourceBreakpoint{{Line: 7}})
	if err != nil {
		t.Fatalf("setBreakpoints: %v", err)
	}
	if len(bps) != 1 {
		t.Fatalf("expected 1 breakpoint, got %d", len(bps))
	}
	if !bps[0].Verified {
		t.Errorf("breakpoint not verified: %s", bps[0].Message)
	}

	if err := client.ConfigurationDone(); err != nil {
		t.Fatalf("configurationDone: %v", err)
	}

	// Wait for stopped at breakpoint
	select {
	case body := <-stopped:
		if body.Reason != "breakpoint" {
			t.Errorf("expected reason breakpoint, got %s", body.Reason)
		}
		t.Logf("stopped: reason=%s threadId=%d", body.Reason, body.ThreadID)

		// Get threads
		threads, err := client.Threads()
		if err != nil {
			t.Fatalf("threads: %v", err)
		}
		if len(threads) == 0 {
			t.Fatal("expected at least 1 thread")
		}
		t.Logf("threads: %d", len(threads))

		// Get stack trace
		frames, err := client.StackTrace(body.ThreadID, 0, 20)
		if err != nil {
			t.Fatalf("stackTrace: %v", err)
		}
		if len(frames) == 0 {
			t.Fatal("expected at least 1 stack frame")
		}
		t.Logf("top frame: %s at %s:%d", frames[0].Name, frames[0].Source.Path, frames[0].Line)

		if frames[0].Line != 7 {
			t.Errorf("expected stopped at line 7, got %d", frames[0].Line)
		}

		// Get scopes
		scopes, err := client.Scopes(frames[0].ID)
		if err != nil {
			t.Fatalf("scopes: %v", err)
		}
		if len(scopes) == 0 {
			t.Fatal("expected at least 1 scope")
		}
		t.Logf("scopes: %d", len(scopes))

		// Get variables (locals)
		var localsRef int
		for _, s := range scopes {
			if s.Name == "Locals" {
				localsRef = s.VariablesReference
				break
			}
		}
		if localsRef > 0 {
			vars, err := client.Variables(localsRef)
			if err != nil {
				t.Fatalf("variables: %v", err)
			}
			t.Logf("variables: %d", len(vars))
			foundX := false
			for _, v := range vars {
				t.Logf("  %s = %s (%s)", v.Name, v.Value, v.Type)
				if v.Name == "x" && v.Value == "42" {
					foundX = true
				}
			}
			if !foundX {
				t.Error("expected to find variable x = 42")
			}
		}

		// Continue to end
		if err := client.Continue(body.ThreadID); err != nil {
			t.Fatalf("continue: %v", err)
		}

	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for stopped event")
	}

	// Wait for program exit (dlv may send exited, terminated, or both)
	select {
	case e := <-exited:
		t.Logf("exited: code=%d", e.ExitCode)
		if e.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", e.ExitCode)
		}
	case <-terminated:
		t.Logf("terminated event received")
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for exited/terminated event")
	}

	// Disconnect
	if err := client.Disconnect(false); err != nil {
		t.Logf("disconnect: %v (may be expected after exit)", err)
	}
}
