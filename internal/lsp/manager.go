package lsp

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/eugenioenko/ttt/internal/config"
)

type Manager struct {
	servers map[string]*Client
	config  config.LSPSettings
	mu      sync.Mutex
}

func NewManager(cfg config.LSPSettings) *Manager {
	return &Manager{
		servers: make(map[string]*Client),
		config:  cfg,
	}
}

func (m *Manager) ClientForLanguage(lang, workDir string) (*Client, error) {
	key := strings.ToLower(lang)

	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.servers[key]; ok {
		return client, nil
	}

	serverCfg, ok := m.config.Servers[key]
	if !ok {
		return nil, fmt.Errorf("no LSP server configured for %q", lang)
	}
	if len(serverCfg.Command) == 0 {
		return nil, fmt.Errorf("empty command for %q", lang)
	}

	slog.Info("lsp starting server", "language", lang, "command", serverCfg.Command)
	client, err := NewClient(serverCfg.Command, workDir)
	if err != nil {
		return nil, fmt.Errorf("start LSP for %s: %w", lang, err)
	}

	rootURI := "file://" + workDir
	if err := client.Initialize(rootURI); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize LSP for %s: %w", lang, err)
	}

	m.servers[key] = client
	return client, nil
}

func (m *Manager) HasServer(lang string) bool {
	key := strings.ToLower(lang)
	_, ok := m.config.Servers[key]
	return ok
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for lang, client := range m.servers {
		slog.Info("lsp shutting down", "language", lang)
		if err := client.Shutdown(); err != nil {
			slog.Debug("lsp shutdown error", "language", lang, "err", err)
			client.Close()
		}
	}
	m.servers = make(map[string]*Client)
}
