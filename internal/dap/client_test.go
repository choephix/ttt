package dap

import (
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/eugenioenko/ttt/internal/jsonrpc"
)

type mockAdapter struct {
	codec  *jsonrpc.Codec
	events []Event
	mu     sync.Mutex
}

func newMockAdapter(r io.Reader, w io.Writer) *mockAdapter {
	return &mockAdapter{codec: jsonrpc.NewCodec(r, w)}
}

func (m *mockAdapter) receive() (Request, error) {
	raw, err := m.codec.Receive()
	if err != nil {
		return Request{}, err
	}
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return Request{}, err
	}
	return req, nil
}

func (m *mockAdapter) respond(reqSeq int, command string, body any) error {
	bodyData, _ := json.Marshal(body)
	return m.codec.Send(Response{
		Seq:        1,
		Type:       "response",
		RequestSeq: reqSeq,
		Success:    true,
		Command:    command,
		Body:       bodyData,
	})
}

func (m *mockAdapter) respondError(reqSeq int, command, message string) error {
	return m.codec.Send(Response{
		Seq:        1,
		Type:       "response",
		RequestSeq: reqSeq,
		Success:    false,
		Command:    command,
		Message:    message,
	})
}

func (m *mockAdapter) sendEvent(event string, body any) error {
	bodyData, _ := json.Marshal(body)
	return m.codec.Send(Event{
		Seq:   1,
		Type:  "event",
		Event: event,
		Body:  bodyData,
	})
}

func newTestClientAndAdapter(t *testing.T) (*Client, *mockAdapter) {
	t.Helper()
	clientRead, adapterWrite := io.Pipe()
	adapterRead, clientWrite := io.Pipe()

	adapter := newMockAdapter(adapterRead, adapterWrite)
	client := &Client{
		codec:   jsonrpc.NewCodec(clientRead, clientWrite),
		seq:     1,
		pending: make(map[int]chan Message),
		done:    make(chan struct{}),
	}
	go client.readLoop()

	t.Cleanup(func() {
		clientWrite.Close()
		adapterWrite.Close()
	})

	return client, adapter
}

func TestInitialize(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, err := adapter.receive()
		if err != nil {
			t.Errorf("adapter receive: %v", err)
			return
		}
		if req.Command != "initialize" {
			t.Errorf("expected initialize, got %s", req.Command)
			return
		}
		adapter.respond(req.Seq, "initialize", Capabilities{
			SupportsConfigurationDoneRequest: true,
			SupportsConditionalBreakpoints:   true,
		})
	}()

	if err := client.Initialize("test"); err != nil {
		t.Fatal(err)
	}

	caps := client.Capabilities()
	if !caps.SupportsConfigurationDoneRequest {
		t.Error("expected SupportsConfigurationDoneRequest")
	}
	if !caps.SupportsConditionalBreakpoints {
		t.Error("expected SupportsConditionalBreakpoints")
	}
}

func TestLaunchAndDisconnect(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, _ := adapter.receive()
		adapter.respond(req.Seq, req.Command, nil)
		req, _ = adapter.receive()
		adapter.respond(req.Seq, req.Command, nil)
	}()

	if err := client.Launch("main.go", false, nil); err != nil {
		t.Fatal("launch:", err)
	}
	if err := client.Disconnect(true); err != nil {
		t.Fatal("disconnect:", err)
	}
}

func TestSetBreakpoints(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, _ := adapter.receive()
		if req.Command != "setBreakpoints" {
			t.Errorf("expected setBreakpoints, got %s", req.Command)
			return
		}
		adapter.respond(req.Seq, req.Command, SetBreakpointsResponseBody{
			Breakpoints: []Breakpoint{
				{ID: 1, Verified: true, Line: 10},
				{ID: 2, Verified: true, Line: 20},
			},
		})
	}()

	bps, err := client.SetBreakpoints("main.go", []SourceBreakpoint{
		{Line: 10},
		{Line: 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bps) != 2 {
		t.Fatalf("expected 2 breakpoints, got %d", len(bps))
	}
	if !bps[0].Verified || bps[0].Line != 10 {
		t.Errorf("breakpoint 0: verified=%v line=%d", bps[0].Verified, bps[0].Line)
	}
	if !bps[1].Verified || bps[1].Line != 20 {
		t.Errorf("breakpoint 1: verified=%v line=%d", bps[1].Verified, bps[1].Line)
	}
}

func TestSteppingSequence(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	commands := make(chan string, 10)
	go func() {
		for {
			req, err := adapter.receive()
			if err != nil {
				return
			}
			commands <- req.Command
			adapter.respond(req.Seq, req.Command, nil)
		}
	}()

	client.Continue(1)
	client.Next(1)
	client.StepIn(1)
	client.StepOut(1)

	got := []string{<-commands, <-commands, <-commands, <-commands}
	expected := []string{"continue", "next", "stepIn", "stepOut"}
	for i, cmd := range got {
		if cmd != expected[i] {
			t.Errorf("step %d: expected %s, got %s", i, expected[i], cmd)
		}
	}
}

func TestThreadsAndStackTrace(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, _ := adapter.receive()
		adapter.respond(req.Seq, req.Command, ThreadsResponseBody{
			Threads: []Thread{
				{ID: 1, Name: "main"},
				{ID: 2, Name: "worker"},
			},
		})
		req, _ = adapter.receive()
		adapter.respond(req.Seq, req.Command, StackTraceResponseBody{
			StackFrames: []StackFrame{
				{ID: 0, Name: "main.main", Source: &Source{Path: "main.go"}, Line: 10},
				{ID: 1, Name: "runtime.main", Source: &Source{Path: "runtime.go"}, Line: 250},
			},
			TotalFrames: 2,
		})
	}()

	threads, err := client.Threads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(threads))
	}

	frames, err := client.StackTrace(1, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0].Name != "main.main" || frames[0].Line != 10 {
		t.Errorf("frame 0: name=%s line=%d", frames[0].Name, frames[0].Line)
	}
}

func TestScopesAndVariables(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, _ := adapter.receive()
		adapter.respond(req.Seq, req.Command, ScopesResponseBody{
			Scopes: []Scope{
				{Name: "Locals", VariablesReference: 100},
				{Name: "Globals", VariablesReference: 200},
			},
		})
		req, _ = adapter.receive()
		adapter.respond(req.Seq, req.Command, VariablesResponseBody{
			Variables: []Variable{
				{Name: "x", Value: "42", Type: "int"},
				{Name: "name", Value: `"hello"`, Type: "string"},
			},
		})
	}()

	scopes, err := client.Scopes(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 || scopes[0].Name != "Locals" {
		t.Fatalf("unexpected scopes: %+v", scopes)
	}

	vars, err := client.Variables(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 || vars[0].Name != "x" || vars[0].Value != "42" {
		t.Fatalf("unexpected variables: %+v", vars)
	}
}

func TestStoppedEvent(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	stopped := make(chan StoppedEventBody, 1)
	client.OnStopped = func(body StoppedEventBody) {
		stopped <- body
	}

	adapter.sendEvent("stopped", StoppedEventBody{
		Reason:            "breakpoint",
		ThreadID:          1,
		AllThreadsStopped: true,
		HitBreakpointIDs:  []int{1},
	})

	body := <-stopped
	if body.Reason != "breakpoint" {
		t.Errorf("expected reason breakpoint, got %s", body.Reason)
	}
	if body.ThreadID != 1 {
		t.Errorf("expected threadId 1, got %d", body.ThreadID)
	}
	if len(body.HitBreakpointIDs) != 1 || body.HitBreakpointIDs[0] != 1 {
		t.Errorf("expected hitBreakpointIds [1], got %v", body.HitBreakpointIDs)
	}
}

func TestOutputEvent(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	output := make(chan OutputEventBody, 1)
	client.OnOutput = func(body OutputEventBody) {
		output <- body
	}

	adapter.sendEvent("output", OutputEventBody{
		Category: "stdout",
		Output:   "hello world\n",
	})

	body := <-output
	if body.Category != "stdout" {
		t.Errorf("expected stdout, got %s", body.Category)
	}
	if body.Output != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", body.Output)
	}
}

func TestExitedEvent(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	exited := make(chan ExitedEventBody, 1)
	client.OnExited = func(body ExitedEventBody) {
		exited <- body
	}

	adapter.sendEvent("exited", ExitedEventBody{ExitCode: 0})

	body := <-exited
	if body.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", body.ExitCode)
	}
}

func TestErrorResponse(t *testing.T) {
	client, adapter := newTestClientAndAdapter(t)

	go func() {
		req, _ := adapter.receive()
		adapter.respondError(req.Seq, req.Command, "program not found")
	}()

	err := client.Launch("nonexistent.go", false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "dap error: program not found" {
		t.Errorf("unexpected error: %v", err)
	}
}
