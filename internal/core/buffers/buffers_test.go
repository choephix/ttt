package buffers

import (
	"ttt/internal/core/buffer"
	"testing"
)

func TestBuffers_AddAndSwitch(t *testing.T) {
	bufs := &Buffers{}
	b1 := &buffer.Buffer{Lines: []string{"a"}}
	b2 := &buffer.Buffer{Lines: []string{"b"}}
	bufs.AddBuffer(b1)
	bufs.AddBuffer(b2)
	if bufs.Active != 1 {
		t.Errorf("expected active=1, got %d", bufs.Active)
	}
	bufs.SwitchBuffer(0)
	if bufs.Current() != b1 {
		t.Error("expected current buffer to be b1")
	}
}
