package jsonrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type Codec struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

func NewCodec(r io.Reader, w io.Writer) *Codec {
	c := &Codec{writer: w}
	if r != nil {
		c.reader = bufio.NewReader(r)
	}
	return c
}

func (c *Codec) Send(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := io.WriteString(c.writer, header); err != nil {
		return err
	}
	_, err = c.writer.Write(data)
	return err
}

func (c *Codec) Receive() (json.RawMessage, error) {
	contentLen := -1
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if val, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("bad Content-Length: %w", err)
			}
			contentLen = n
		}
	}
	if contentLen < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLen)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}
