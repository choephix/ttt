package term

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// CollectPasteText reconstructs text from a sequence of tcell key events
// delivered during a bracketed paste. tcell delivers \r as KeyEnter and
// \n as KeyCtrlJ; CRLF line endings produce both in sequence. This function
// normalizes all line endings to \n.
func CollectPasteText(events []*tcell.EventKey) string {
	var buf strings.Builder
	for _, ev := range events {
		switch ev.Key() {
		case tcell.KeyRune:
			buf.WriteRune(KeyRune(ev))
		case tcell.KeyEnter:
			buf.WriteRune('\r')
		case tcell.KeyCtrlJ:
			buf.WriteRune('\n')
		case tcell.KeyTab:
			buf.WriteRune('\t')
		}
	}
	text := buf.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}
