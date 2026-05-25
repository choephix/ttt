package view

import (
	"time"

	"github.com/eugenioenko/ttt/internal/term"
)

type NotifyLevel int

const (
	NotifyInfo NotifyLevel = iota
	NotifyWarning
	NotifyError
)

func (l NotifyLevel) Style() term.Style {
	switch l {
	case NotifyWarning:
		return term.StyleWarning
	case NotifyError:
		return term.StyleDanger
	default:
		return term.StyleStatusBar
	}
}

type StatusBar struct {
	FileName     string
	Line         int
	Col          int
	Dirty        bool
	Branch       string
	Blame        string
	DiagMessage  string
	DiagLevel    NotifyLevel
	Language     string
	LSP          bool
	TabSize      int
	Notification string
	NotifyLevel  NotifyLevel
	NotifyExpiry time.Time
}

func (s *StatusBar) SetNotification(msg string, level NotifyLevel, duration time.Duration) {
	s.Notification = msg
	s.NotifyLevel = level
	s.NotifyExpiry = time.Now().Add(duration)
}

func (s *StatusBar) DismissNotification() {
	s.Notification = ""
	s.NotifyExpiry = time.Time{}
}

func (s *StatusBar) IsNotificationActive() bool {
	if s.Notification == "" {
		return false
	}
	if !s.NotifyExpiry.IsZero() && time.Now().After(s.NotifyExpiry) {
		s.DismissNotification()
		return false
	}
	return true
}
