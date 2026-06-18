package terminal

import (
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/eugenioenko/vt10x"
)

const (
	AttrReverse   int16 = 1
	AttrUnderline int16 = 2
	AttrBold      int16 = 4
	AttrItalic    int16 = 16
	AttrBlink     int16 = 32
)

type Terminal struct {
	mu         sync.Mutex
	vt         vt10x.Terminal
	ptm        *os.File
	cmd        *exec.Cmd
	cols, rows int
	done       chan struct{}
	closed     bool
	exited     bool
	OnUpdate   func()
	OnExit     func()
}

func New(shell string, cols, rows, scrollbackMax int, env []string, dir string) (*Terminal, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}
	if scrollbackMax <= 0 {
		scrollbackMax = 1000
	}

	t := &Terminal{
		cols: cols,
		rows: rows,
		done: make(chan struct{}),
	}

	t.vt = vt10x.New(vt10x.WithSize(cols, rows), vt10x.WithScrollback(scrollbackMax))

	cmd := exec.Command(shell)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	ptm, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	t.ptm = ptm
	t.cmd = cmd

	pty.Setsize(ptm, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})

	go t.readLoop()

	return t, nil
}

func (t *Terminal) readLoop() {
	defer close(t.done)
	buf := make([]byte, 4096)
	for {
		n, err := t.ptm.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.vt.Write(buf[:n])
			t.mu.Unlock()
			if t.OnUpdate != nil {
				t.OnUpdate()
			}
		}
		if err != nil {
			t.mu.Lock()
			t.exited = true
			t.mu.Unlock()
			if t.OnExit != nil {
				t.OnExit()
			}
			return
		}
	}
}

func (t *Terminal) WriteString(s string) {
	io.WriteString(t.ptm, s)
}

func (t *Terminal) Resize(cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cols = cols
	t.rows = rows
	t.vt.Resize(cols, rows)
	pty.Setsize(t.ptm, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func (t *Terminal) Snapshot(fn func(view vt10x.View)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fn(t.vt)
}

func (t *Terminal) CursorPos() (x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	c := t.vt.Cursor()
	return c.X, c.Y
}

func (t *Terminal) Size() (cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cols, t.rows
}

func (t *Terminal) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()

	t.ptm.Close()
	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
	<-t.done
}

func (t *Terminal) ScrollbackLen() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vt.ScrollbackLen()
}

func (t *Terminal) Mode() vt10x.ModeFlag {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vt.Mode()
}
