package dap

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type AdapterConfig struct {
	Command []string `json:"command"`
}

type Manager struct {
	adapters map[string]AdapterConfig
	client   *Client
	mu       sync.Mutex

	OnStopped    func(body StoppedEventBody)
	OnContinued  func(body ContinuedEventBody)
	OnExited     func(body ExitedEventBody)
	OnTerminated func(body TerminatedEventBody)
	OnThread     func(body ThreadEventBody)
	OnOutput     func(body OutputEventBody)
	OnBreakpoint func(body BreakpointEventBody)
}

func NewManager(adapters map[string]AdapterConfig) *Manager {
	return &Manager{
		adapters: adapters,
	}
}

func (m *Manager) HasAdapter(language string) bool {
	_, ok := m.adapters[language]
	return ok
}

func (m *Manager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client != nil
}

func (m *Manager) Client() *Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client
}

func (m *Manager) Start(language, workDir, program string) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return nil, fmt.Errorf("debug session already active")
	}

	cfg, ok := m.adapters[language]
	if !ok {
		return nil, fmt.Errorf("no debug adapter configured for %q", language)
	}
	if len(cfg.Command) == 0 {
		return nil, fmt.Errorf("empty command for debug adapter %q", language)
	}

	slog.Info("dap starting adapter", "language", language, "command", cfg.Command)
	client, err := NewClient(cfg.Command, workDir)
	if err != nil {
		return nil, fmt.Errorf("start debug adapter for %s: %w", language, err)
	}

	client.OnStopped = m.OnStopped
	client.OnContinued = m.OnContinued
	client.OnExited = m.OnExited
	client.OnTerminated = m.OnTerminated
	client.OnThread = m.OnThread
	client.OnOutput = m.OnOutput
	client.OnBreakpoint = m.OnBreakpoint

	if err := client.Initialize(language); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize debug adapter for %s: %w", language, err)
	}

	m.client = client
	return client, nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	client := m.client
	m.client = nil
	m.mu.Unlock()

	if client == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := client.Disconnect(true); err != nil {
			slog.Debug("dap disconnect error", "err", err)
			client.Close()
			return
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		slog.Debug("dap shutdown timeout, killing")
		client.Close()
	}
}
