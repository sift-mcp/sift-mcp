package mcp

import (
	"context"
	"encoding/json"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "sift-mcp"
	ServerVersion   = "0.1.0"
)

type ToolFunc func(ctx context.Context, args map[string]interface{}) ToolsCallResult

type Handler struct {
	tools    map[string]Tool
	handlers map[string]ToolFunc
	readOnly bool
}

func NewHandler(readOnly bool) *Handler {
	return &Handler{
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolFunc),
		readOnly: readOnly,
	}
}

func (h *Handler) RegisterTool(tool Tool, handler ToolFunc) {
	h.tools[tool.Name] = tool
	h.handlers[tool.Name] = handler
}

func (h *Handler) Handle(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req)
	case "initialized":
		return h.handleInitialized(req)
	case "tools/list":
		return h.handleToolsList(req)
	case "tools/call":
		return h.handleToolsCall(ctx, req)
	case "ping":
		return h.handlePing(req)
	default:
		return &Response{
			JSONRPC: "2.0",
			Error:   NewError(ErrorMethodNotFound, "Method not found: "+req.Method),
			ID:      req.ID,
		}
	}
}

func (h *Handler) handleInitialize(req *Request) *Response {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: false},
		},
		ServerInfo: ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func (h *Handler) handleInitialized(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result:  struct{}{},
		ID:      req.ID,
	}
}

func (h *Handler) handleToolsList(req *Request) *Response {
	tools := make([]Tool, 0, len(h.tools))
	for _, tool := range h.tools {
		tools = append(tools, tool)
	}

	return &Response{
		JSONRPC: "2.0",
		Result:  ToolsListResult{Tools: tools},
		ID:      req.ID,
	}
}

func (h *Handler) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params ToolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error:   NewError(ErrorInvalidParams, "Invalid params: "+err.Error()),
			ID:      req.ID,
		}
	}

	handler, exists := h.handlers[params.Name]
	if !exists {
		return &Response{
			JSONRPC: "2.0",
			Error:   NewError(ErrorInvalidParams, "Unknown tool: "+params.Name),
			ID:      req.ID,
		}
	}

	result := handler(ctx, params.Arguments)

	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func (h *Handler) handlePing(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result:  struct{}{},
		ID:      req.ID,
	}
}

func (h *Handler) IsReadOnly() bool {
	return h.readOnly
}
