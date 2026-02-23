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

type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
)

type ServerConfig struct {
	Transport TransportType
	ReadOnly  bool
	Port      int
}

type Server struct {
	config  ServerConfig
	handler *Handler
	mu      sync.RWMutex
	running bool
}

func NewServer(handler *Handler, cfg ServerConfig) *Server {
	return &Server{
		config:  cfg,
		handler: handler,
	}
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

	switch s.config.Transport {
	case TransportStdio:
		return s.serveStdio(ctx)
	case TransportHTTP:
		return s.serveHTTP(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
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
			s.writeResponse(writer, resp)
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

func (s *Server) serveHTTP(ctx context.Context) error {
	return fmt.Errorf("HTTP+SSE transport not yet implemented")
}

func (s *Server) IsReadOnly() bool {
	return s.config.ReadOnly
}
