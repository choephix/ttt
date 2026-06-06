package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

func TestCodecSend(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(nil, &buf)

	msg := map[string]any{"method": "test", "id": 1}
	if err := codec.Send(msg); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()
	if raw[:16] != "Content-Length: " {
		t.Fatalf("expected Content-Length header, got %q", raw[:16])
	}

	headerEnd := bytes.Index(buf.Bytes(), []byte("\r\n\r\n"))
	if headerEnd < 0 {
		t.Fatal("missing header separator")
	}
	body := buf.Bytes()[headerEnd+4:]

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "test" {
		t.Fatalf("expected method test, got %v", got["method"])
	}
}

func TestCodecReceive(t *testing.T) {
	body := `{"id":1,"result":"ok"}`
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	codec := NewCodec(bytes.NewReader([]byte(msg)), nil)

	raw, err := codec.Receive()
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["result"] != "ok" {
		t.Fatalf("expected result ok, got %v", got["result"])
	}
}

func TestCodecRoundTrip(t *testing.T) {
	pr, pw := io.Pipe()
	sender := NewCodec(nil, pw)
	receiver := NewCodec(pr, nil)

	go func() {
		sender.Send(map[string]string{"method": "hello"})
		pw.Close()
	}()

	raw, err := receiver.Receive()
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "hello" {
		t.Fatalf("expected hello, got %s", got["method"])
	}
}

func TestCodecReceiveMissingContentLength(t *testing.T) {
	msg := "Content-Type: application/json\r\n\r\n{}"
	codec := NewCodec(bytes.NewReader([]byte(msg)), nil)

	_, err := codec.Receive()
	if err == nil {
		t.Fatal("expected error for missing Content-Length")
	}
}
