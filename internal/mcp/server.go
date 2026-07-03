package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type Server struct {
	handler *Handler
	mu      sync.RWMutex
	running bool
}

func NewServer(handler *Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	return s.serveStdio(ctx)
}

type readResult struct {
	line []byte
	err  error
}

func (s *Server) serveStdio(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	lineChan := make(chan readResult, 1)

	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			lineChan <- readResult{line: line, err: err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-lineChan:
			if result.err != nil {
				if result.err == io.EOF {
					return nil
				}
				return fmt.Errorf("read error: %w", result.err)
			}

			if len(result.line) == 0 {
				continue
			}

			var req Request
			if err := json.Unmarshal(result.line, &req); err != nil {
				s.writeResponse(writer, &Response{
					JSONRPC: "2.0",
					Error:   NewError(ErrorParseError, "Parse error"),
					ID:      nil,
				})
				continue
			}

			resp := s.handler.Handle(ctx, &req)
			if resp != nil {
				s.writeResponse(writer, resp)
			}
		}
	}
}

func (s *Server) writeResponse(w *bufio.Writer, resp *Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	w.Write(data)
	w.WriteByte('\n')
	w.Flush()
}
