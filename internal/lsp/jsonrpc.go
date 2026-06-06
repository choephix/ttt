package lsp

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/eugenioenko/ttt/internal/jsonrpc"
)

type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (r *Response) IsNotification() bool {
	return r.ID == nil && r.Method != ""
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

type Codec struct {
	codec *jsonrpc.Codec
}

func NewCodec(r io.Reader, w io.Writer) *Codec {
	return &Codec{codec: jsonrpc.NewCodec(r, w)}
}

func (c *Codec) Send(req Request) error {
	return c.codec.Send(req)
}

func (c *Codec) Receive() (Response, error) {
	raw, err := c.codec.Receive()
	if err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}
