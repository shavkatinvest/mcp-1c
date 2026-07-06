package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPostDocumentHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document/post" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","posted":true,"date":"2026-07-02T10:00:00"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewPostDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":            "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "post_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "posted=true") {
		t.Errorf("expected posted=true in text, got:\n%s", text)
	}
}

func TestPostDocumentHandler_MissingRef(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewPostDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{"document_type": "РеализацияТоваровУслуг"})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "post_document", Arguments: args}}

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing ref")
	}
}

func TestPostDocumentHandler_StructuredFailure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"posting_failed","message":"Отрицательный остаток на складе"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewPostDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":            "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "post_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for posting_failed")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Отрицательный остаток") {
		t.Errorf("expected 1C message verbatim, got:\n%s", text)
	}
}

func TestUnpostDocumentHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document/unpost" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","posted":false,"date":"2026-07-02T10:00:00"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewUnpostDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":            "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "unpost_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "posted=false") {
		t.Errorf("expected posted=false in text, got:\n%s", text)
	}
}

func TestPostDocumentTool(t *testing.T) {
	tool := PostDocumentTool()
	if tool.Name != "post_document" {
		t.Errorf("expected post_document, got %s", tool.Name)
	}
}

func TestUnpostDocumentTool(t *testing.T) {
	tool := UnpostDocumentTool()
	if tool.Name != "unpost_document" {
		t.Errorf("expected unpost_document, got %s", tool.Name)
	}
}
