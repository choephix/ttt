package plugin

import (
	"fmt"
	"testing"
)

type mockSystemAPI struct {
	execBinary string
	execArgs   []string
	stdout     string
	stderr     string
	exitCode   int
	execErr    error
	envVars    map[string]string
}

func (m *mockSystemAPI) Exec(binary string, args []string) (string, string, int, error) {
	m.execBinary = binary
	m.execArgs = args
	return m.stdout, m.stderr, m.exitCode, m.execErr
}

func (m *mockSystemAPI) Env(name string) string {
	if m.envVars != nil {
		return m.envVars[name]
	}
	return ""
}

func setupTestPluginWithSystem(perms PermissionSet, sys *mockSystemAPI) (*Plugin, func()) {
	p, cleanup := newTestPluginBase(perms)
	p.System = sys
	return p, cleanup
}

func TestSystemExec(t *testing.T) {
	mock := &mockSystemAPI{stdout: "container1\ncontainer2\n", exitCode: 0}
	p, cleanup := setupTestPluginWithSystem(
		PermissionSet{SystemExec: []string{"docker"}},
		mock,
	)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		local result = sys.exec("docker", {"ps", "--format", "{{.Names}}"})
		_G.stdout = result.stdout
		_G.exit_code = result.exit_code
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if p.State.GetGlobal("stdout").String() != "container1\ncontainer2\n" {
		t.Errorf("unexpected stdout: %q", p.State.GetGlobal("stdout").String())
	}
	if mock.execBinary != "docker" {
		t.Errorf("expected binary 'docker', got %q", mock.execBinary)
	}
	if len(mock.execArgs) != 3 || mock.execArgs[0] != "ps" {
		t.Errorf("unexpected args: %v", mock.execArgs)
	}
}

func TestSystemExecDeniedBinary(t *testing.T) {
	mock := &mockSystemAPI{}
	p, cleanup := setupTestPluginWithSystem(
		PermissionSet{SystemExec: []string{"docker"}},
		mock,
	)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		sys.exec("rm", {"-rf", "/"})
	`)
	if err == nil {
		t.Fatal("expected error for unapproved binary")
	}
}

func TestSystemExecError(t *testing.T) {
	mock := &mockSystemAPI{execErr: fmt.Errorf("command not found")}
	p, cleanup := setupTestPluginWithSystem(
		PermissionSet{SystemExec: []string{"missing"}},
		mock,
	)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		local result = sys.exec("missing", {})
		_G.stderr = result.stderr
		_G.code = result.exit_code
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if p.State.GetGlobal("code").String() != "-1" {
		t.Errorf("expected exit code -1, got %s", p.State.GetGlobal("code").String())
	}
}

func TestSystemEnv(t *testing.T) {
	mock := &mockSystemAPI{envVars: map[string]string{"HOME": "/home/test"}}
	p, cleanup := setupTestPluginWithSystem(
		PermissionSet{SystemEnv: true},
		mock,
	)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		_G.home = sys.env("HOME")
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if p.State.GetGlobal("home").String() != "/home/test" {
		t.Errorf("expected '/home/test', got %q", p.State.GetGlobal("home").String())
	}
}

func TestSystemNoExecPermission(t *testing.T) {
	mock := &mockSystemAPI{}
	p, cleanup := setupTestPluginWithSystem(PermissionSet{}, mock)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		sys.exec("ls", {})
	`)
	if err == nil {
		t.Fatal("expected error when system.exec not granted")
	}
}

func TestSystemNoEnvPermission(t *testing.T) {
	mock := &mockSystemAPI{}
	p, cleanup := setupTestPluginWithSystem(PermissionSet{}, mock)
	defer cleanup()

	err := p.State.DoString(`
		local sys = require("ttt.system")
		sys.env("HOME")
	`)
	if err == nil {
		t.Fatal("expected error when system.env not granted")
	}
}
