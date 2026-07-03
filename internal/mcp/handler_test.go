package mcp

import (
	"context"
	"testing"
)

func newTestHandler() *Handler {
	return NewHandler()
}

func TestHandleNotificationReturnsNoResponse(t *testing.T) {
	h := newTestHandler()

	resp := h.Handle(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})

	if resp != nil {
		t.Fatalf("expected no response to notification, got %+v", resp)
	}
}

func TestHandleUnknownNotificationReturnsNoResponse(t *testing.T) {
	h := newTestHandler()

	resp := h.Handle(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
	})

	if resp != nil {
		t.Fatalf("expected no response to unknown notification, got %+v", resp)
	}
}

func TestHandleRequestWithIDReturnsResponse(t *testing.T) {
	h := newTestHandler()

	resp := h.Handle(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "ping",
		ID:      float64(1),
	})

	if resp == nil {
		t.Fatal("expected response to ping request, got nil")
	}
	if resp.ID != float64(1) {
		t.Errorf("expected response ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %+v", resp.Error)
	}
}

func TestHandleUnknownMethodWithIDReturnsError(t *testing.T) {
	h := newTestHandler()

	resp := h.Handle(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "does/not/exist",
		ID:      float64(2),
	})

	if resp == nil {
		t.Fatal("expected error response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response, got none")
	}
	if resp.Error.Code != ErrorMethodNotFound {
		t.Errorf("expected error code %d, got %d", ErrorMethodNotFound, resp.Error.Code)
	}
}

func TestHandleInitializeReturnsServerInfo(t *testing.T) {
	h := newTestHandler()

	resp := h.Handle(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      float64(3),
	})

	if resp == nil {
		t.Fatal("expected response to initialize, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("expected no error, got %+v", resp.Error)
	}
	result, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}
	if result.ServerInfo.Name != ServerName {
		t.Errorf("expected server name %q, got %q", ServerName, result.ServerInfo.Name)
	}
}

func TestIsNotification(t *testing.T) {
	withID := &Request{JSONRPC: "2.0", Method: "ping", ID: float64(1)}
	if withID.IsNotification() {
		t.Error("request with ID should not be a notification")
	}

	withoutID := &Request{JSONRPC: "2.0", Method: "notifications/initialized"}
	if !withoutID.IsNotification() {
		t.Error("request without ID should be a notification")
	}
}
