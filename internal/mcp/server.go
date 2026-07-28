// Package mcp implements a Model Context Protocol server over stdio.
//
// The server reads JSON-RPC 2.0 messages from stdin (Content-Length delimited)
// and writes responses to stdout. This allows AI agents to use jobsearch
// commands as MCP tools.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// RawMessage is a JSON-RPC 2.0 message.
type RawMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Server handles MCP protocol messages.
type Server struct {
	in      io.Reader
	out     io.Writer
	logErr  io.Writer
	handler Handler
}

// Handler processes MCP method calls.
type Handler interface {
	HandleMethod(method string, params json.RawMessage) (any, *RPCError)
}

// New creates an MCP server that reads from in and writes to out.
func New(in io.Reader, out io.Writer, handler Handler) *Server {
	return &Server{
		in:      in,
		out:     out,
		logErr:  os.Stderr,
		handler: handler,
	}
}

// Run starts the MCP server read loop. It blocks until stdin closes.
func (s *Server) Run() error {
	br := bufio.NewReader(s.in)
	for {
		msg, err := s.readMessage(br)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		s.handleMessage(msg)
	}
}

// readMessage reads a Content-Length delimited message.
func (s *Server) readMessage(br *bufio.Reader) (RawMessage, error) {
	var contentLen int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return RawMessage{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// End of headers
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			n, err := strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
			if err != nil {
				return RawMessage{}, fmt.Errorf("bad Content-Length: %s", line)
			}
			if contentLen != 0 {
				return RawMessage{}, fmt.Errorf("duplicate Content-Length")
			}
			contentLen = n
		}
	}

	if contentLen == 0 {
		return RawMessage{}, fmt.Errorf("missing Content-Length header")
	}

	body := make([]byte, contentLen)
	if _, err := io.ReadFull(br, body); err != nil {
		return RawMessage{}, fmt.Errorf("read body: %w", err)
	}

	var msg RawMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return RawMessage{}, fmt.Errorf("parse JSON: %w", err)
	}
	return msg, nil
}

// handleMessage processes a single message and sends the response.
func (s *Server) handleMessage(msg RawMessage) {
	// Notifications (no ID) — no response
	if len(msg.ID) == 0 || string(msg.ID) == "null" {
		s.handleNotification(msg)
		return
	}

	result, rpcErr := s.handler.HandleMethod(msg.Method, msg.Params)

	resp := RawMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			resp.Error = &RPCError{Code: -32603, Message: "internal error: marshal result"}
		} else {
			resp.Result = data
		}
	}

	s.writeMessage(resp)
}

func (s *Server) handleNotification(msg RawMessage) {
	switch msg.Method {
	case "notifications/initialized":
		// No-op, client confirmed initialization
	default:
		log.Printf("[mcp] unhandled notification: %s", msg.Method)
	}
}

// writeMessage writes a JSON-RPC message with Content-Length headers.
func (s *Server) writeMessage(msg RawMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[mcp] marshal error: %s", err)
		return
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(s.out, header); err != nil {
		log.Printf("[mcp] write header error: %s", err)
		return
	}
	if _, err := s.out.Write(data); err != nil {
		log.Printf("[mcp] write body error: %s", err)
	}
}

// MethodRequest is a parsed request for a specific method.
type MethodRequest struct {
	Method string
	ID     json.RawMessage
}
