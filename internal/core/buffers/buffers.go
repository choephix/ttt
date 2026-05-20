package buffers

import "ttt/internal/core/buffer"

// Buffers manages multiple open buffers.
type Buffers struct {
	List   []*buffer.Buffer
	Active int // index of the active buffer
}

// AddBuffer adds a new buffer and makes it active.
func (b *Buffers) AddBuffer(buf *buffer.Buffer) {
	b.List = append(b.List, buf)
	b.Active = len(b.List) - 1
}

// SwitchBuffer switches to the buffer at the given index.
func (b *Buffers) SwitchBuffer(idx int) {
	if idx >= 0 && idx < len(b.List) {
		b.Active = idx
	}
}

// Current returns the active buffer.
func (b *Buffers) Current() *buffer.Buffer {
	if b.Active >= 0 && b.Active < len(b.List) {
		return b.List[b.Active]
	}
	return nil
}
