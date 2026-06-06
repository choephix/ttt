package dap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/eugenioenko/ttt/internal/jsonrpc"
)

type Client struct {
	cmd     *exec.Cmd
	conn    io.Closer
	codec   *jsonrpc.Codec
	seq     int
	pending map[int]chan Message
	mu      sync.Mutex
	done    chan struct{}

	capabilities Capabilities

	OnStopped    func(body StoppedEventBody)
	OnContinued  func(body ContinuedEventBody)
	OnExited     func(body ExitedEventBody)
	OnTerminated func(body TerminatedEventBody)
	OnThread     func(body ThreadEventBody)
	OnOutput     func(body OutputEventBody)
	OnBreakpoint func(body BreakpointEventBody)
}

func NewClient(command []string, workDir string) (*Client, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = workDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", command[0], err)
	}

	c := &Client{
		cmd:     cmd,
		codec:   jsonrpc.NewCodec(stdout, stdin),
		seq:     1,
		pending: make(map[int]chan Message),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func NewTCPClient(command []string, workDir string) (*Client, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", command[0], err)
	}

	addr, err := parseListenAddr(stdout)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("parse listen address: %w", err)
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}

	c := &Client{
		cmd:     cmd,
		conn:    conn,
		codec:   jsonrpc.NewCodec(conn, conn),
		seq:     1,
		pending: make(map[int]chan Message),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func parseListenAddr(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	deadline := time.After(10 * time.Second)
	result := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if idx := strings.Index(line, "listening at: "); idx >= 0 {
				result <- line[idx+len("listening at: "):]
				return
			}
		}
		errCh <- fmt.Errorf("adapter exited without reporting listen address")
	}()

	select {
	case addr := <-result:
		return addr, nil
	case err := <-errCh:
		return "", err
	case <-deadline:
		return "", fmt.Errorf("timeout waiting for listen address")
	}
}

func (c *Client) readLoop() {
	defer close(c.done)
	for {
		raw, err := c.codec.Receive()
		if err != nil {
			slog.Debug("dap read loop exit", "err", err)
			c.mu.Lock()
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int]chan Message)
			c.mu.Unlock()
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			slog.Debug("dap unmarshal error", "err", err)
			continue
		}

		if msg.IsResponse() {
			c.mu.Lock()
			ch, ok := c.pending[msg.RequestSeq]
			if ok {
				delete(c.pending, msg.RequestSeq)
			}
			c.mu.Unlock()
			if ok {
				ch <- msg
			}
		} else if msg.IsEvent() {
			c.handleEvent(msg)
		}
	}
}

func (c *Client) handleEvent(msg Message) {
	slog.Debug("dap event", "event", msg.Event)
	switch msg.Event {
	case "stopped":
		if c.OnStopped != nil {
			var body StoppedEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnStopped(body)
		}
	case "continued":
		if c.OnContinued != nil {
			var body ContinuedEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnContinued(body)
		}
	case "exited":
		if c.OnExited != nil {
			var body ExitedEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnExited(body)
		}
	case "terminated":
		if c.OnTerminated != nil {
			var body TerminatedEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnTerminated(body)
		}
	case "thread":
		if c.OnThread != nil {
			var body ThreadEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnThread(body)
		}
	case "output":
		if c.OnOutput != nil {
			var body OutputEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnOutput(body)
		}
	case "breakpoint":
		if c.OnBreakpoint != nil {
			var body BreakpointEventBody
			json.Unmarshal(msg.Body, &body)
			c.OnBreakpoint(body)
		}
	case "initialized":
		// handled inline during Initialize()
	}
}

func (c *Client) send(command string, arguments any) (Message, error) {
	c.mu.Lock()
	seq := c.seq
	c.seq++
	ch := make(chan Message, 1)
	c.pending[seq] = ch
	c.mu.Unlock()

	req := Request{
		Seq:     seq,
		Type:    "request",
		Command: command,
	}
	if arguments != nil {
		data, err := json.Marshal(arguments)
		if err != nil {
			c.mu.Lock()
			delete(c.pending, seq)
			c.mu.Unlock()
			return Message{}, err
		}
		req.Arguments = data
	}

	if err := c.codec.Send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
		return Message{}, err
	}

	resp, ok := <-ch
	if !ok {
		return Message{}, fmt.Errorf("connection closed")
	}
	if !resp.Success {
		return resp, fmt.Errorf("dap error: %s", resp.Message)
	}
	return resp, nil
}

func (c *Client) Initialize(adapterID string) error {
	resp, err := c.send("initialize", InitializeRequestArguments{
		ClientID:        "ttt",
		ClientName:      "ttt",
		AdapterID:       adapterID,
		LinesStartAt1:   true,
		ColumnsStartAt1: true,
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if len(resp.Body) > 0 {
		json.Unmarshal(resp.Body, &c.capabilities)
	}
	slog.Debug("dap initialized", "capabilities", c.capabilities)
	return nil
}

func (c *Client) Capabilities() Capabilities {
	return c.capabilities
}

func (c *Client) Launch(program string, noDebug bool, extra map[string]any) error {
	args := map[string]any{
		"program": program,
		"noDebug": noDebug,
	}
	for k, v := range extra {
		args[k] = v
	}
	_, err := c.send("launch", args)
	return err
}

func (c *Client) Attach(args map[string]any) error {
	_, err := c.send("attach", args)
	return err
}

func (c *Client) ConfigurationDone() error {
	_, err := c.send("configurationDone", nil)
	return err
}

func (c *Client) SetBreakpoints(path string, breakpoints []SourceBreakpoint) ([]Breakpoint, error) {
	resp, err := c.send("setBreakpoints", SetBreakpointsArguments{
		Source:      Source{Path: path},
		Breakpoints: breakpoints,
	})
	if err != nil {
		return nil, err
	}
	var body SetBreakpointsResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse setBreakpoints response: %w", err)
	}
	return body.Breakpoints, nil
}

func (c *Client) Continue(threadID int) error {
	_, err := c.send("continue", ContinueArguments{ThreadID: threadID})
	return err
}

func (c *Client) Next(threadID int) error {
	_, err := c.send("next", StepArguments{ThreadID: threadID})
	return err
}

func (c *Client) StepIn(threadID int) error {
	_, err := c.send("stepIn", StepArguments{ThreadID: threadID})
	return err
}

func (c *Client) StepOut(threadID int) error {
	_, err := c.send("stepOut", StepArguments{ThreadID: threadID})
	return err
}

func (c *Client) Pause(threadID int) error {
	_, err := c.send("pause", PauseArguments{ThreadID: threadID})
	return err
}

func (c *Client) Threads() ([]Thread, error) {
	resp, err := c.send("threads", nil)
	if err != nil {
		return nil, err
	}
	var body ThreadsResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse threads response: %w", err)
	}
	return body.Threads, nil
}

func (c *Client) StackTrace(threadID, startFrame, levels int) ([]StackFrame, error) {
	resp, err := c.send("stackTrace", StackTraceArguments{
		ThreadID:   threadID,
		StartFrame: startFrame,
		Levels:     levels,
	})
	if err != nil {
		return nil, err
	}
	var body StackTraceResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse stackTrace response: %w", err)
	}
	return body.StackFrames, nil
}

func (c *Client) Scopes(frameID int) ([]Scope, error) {
	resp, err := c.send("scopes", ScopesArguments{FrameID: frameID})
	if err != nil {
		return nil, err
	}
	var body ScopesResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse scopes response: %w", err)
	}
	return body.Scopes, nil
}

func (c *Client) Variables(variablesRef int) ([]Variable, error) {
	resp, err := c.send("variables", VariablesArguments{
		VariablesReference: variablesRef,
	})
	if err != nil {
		return nil, err
	}
	var body VariablesResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse variables response: %w", err)
	}
	return body.Variables, nil
}

func (c *Client) Evaluate(expression string, frameID int, context string) (*EvaluateResponseBody, error) {
	resp, err := c.send("evaluate", EvaluateArguments{
		Expression: expression,
		FrameID:    frameID,
		Context:    context,
	})
	if err != nil {
		return nil, err
	}
	var body EvaluateResponseBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse evaluate response: %w", err)
	}
	return &body, nil
}

func (c *Client) Disconnect(terminateDebuggee bool) error {
	_, err := c.send("disconnect", DisconnectArguments{
		TerminateDebuggee: terminateDebuggee,
	})
	return err
}

func (c *Client) Terminate() error {
	_, err := c.send("terminate", nil)
	return err
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.cmd.Process.Kill()
	c.cmd.Wait()
}
