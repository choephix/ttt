package ui

type WindowManager struct {
	Windows []*Window
	Focus   int // index of focused window
}

func (m *WindowManager) AddWindow(w *Window) {
	m.Windows = append(m.Windows, w)
	m.Focus = len(m.Windows) - 1
}

func (m *WindowManager) SwitchFocus(idx int) {
	if idx >= 0 && idx < len(m.Windows) {
		m.Focus = idx
	}
}

func (m *WindowManager) Current() *Window {
	if m.Focus >= 0 && m.Focus < len(m.Windows) {
		return m.Windows[m.Focus]
	}
	return nil
}
