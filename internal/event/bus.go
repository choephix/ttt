package event

type EventType string

const (
	BufferChanged    EventType = "buffer.changed"
	BufferDirty      EventType = "buffer.dirty"
	FileOpened       EventType = "file.opened"
	FileSaved        EventType = "file.saved"
	FileClosed       EventType = "file.closed"
	CursorMoved      EventType = "cursor.moved"
	SelectionChanged EventType = "selection.changed"
	FocusChanged     EventType = "focus.changed"
	ThemeChanged     EventType = "theme.changed"
	ConfigChanged    EventType = "config.changed"
	LayoutChanged    EventType = "layout.changed"
)

type Event struct {
	Type    EventType
	Payload any
}

type Bus struct {
	subscribers map[EventType][]func(Event)
}

func NewBus() *Bus {
	return &Bus{subscribers: make(map[EventType][]func(Event))}
}

func (b *Bus) Subscribe(t EventType, handler func(Event)) {
	b.subscribers[t] = append(b.subscribers[t], handler)
}

func (b *Bus) Publish(e Event) {
	for _, handler := range b.subscribers[e.Type] {
		handler(e)
	}
}
