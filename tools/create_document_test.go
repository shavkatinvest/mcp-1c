package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCreateDocumentHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		var req onec.CreateDocumentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshaling body: %v", err)
		}
		if req.Type != "РеализацияТоваровУслуг" {
			t.Errorf("expected type РеализацияТоваровУслуг, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","number":"ЗН-000001","date":"2026-07-02T10:00:00","posted":false}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"attributes":    map[string]any{"Организация": "22222222-2222-2222-2222-222222222222"},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "create_document", Arguments: args},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result: %v", result.Content)
	}

	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	for _, want := range []string{"11111111-1111-1111-1111-111111111111", "ЗН-000001", "НЕ проведён", "post_document"} {
		if !strings.Contains(tc.Text, want) {
			t.Errorf("expected text to contain %q, got:\n%s", want, tc.Text)
		}
	}
}

func TestCreateDocumentHandler_MissingType(t *testing.T) {
	httpCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled = true
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "create_document", Arguments: args},
	}

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing document_type")
	}
	if httpCalled {
		t.Error("HTTP call should not have been made without document_type")
	}
}

func TestCreateDocumentHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"unknown_document_type","message":"Тип документа 'Foo' не найден","field":"type"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{"document_type": "Foo"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "create_document", Arguments: args},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"unknown_document_type", "Тип документа 'Foo' не найден", "type"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in error text, got:\n%s", want, text)
		}
	}
}

func TestCreateDocumentTool(t *testing.T) {
	tool := CreateDocumentTool()
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
	if tool.Name != "create_document" {
		t.Errorf("expected tool name create_document, got %s", tool.Name)
	}
	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
