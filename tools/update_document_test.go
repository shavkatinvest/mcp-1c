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

func TestUpdateDocumentHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document/update" {
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
		var req onec.UpdateObjectRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshaling body: %v", err)
		}
		if req.Type != "РеализацияТоваровУслуг" {
			t.Errorf("expected type РеализацияТоваровУслуг, got %s", req.Type)
		}
		if req.Ref != "11111111-1111-1111-1111-111111111111" {
			t.Errorf("unexpected ref: %s", req.Ref)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ref":"11111111-1111-1111-1111-111111111111",
			"type":"РеализацияТоваровУслуг",
			"number":"ЗН-000001",
			"date":"2026-07-02T10:00:00",
			"posted":false,
			"deletion_mark":false,
			"attributes":{},
			"tabular_sections":{}
		}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewUpdateDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":           "11111111-1111-1111-1111-111111111111",
		"attributes":    map[string]any{"Комментарий": "изменено"},
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result: %v", result.Content)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"изменён", "11111111-1111-1111-1111-111111111111", "ЗН-000001"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestUpdateDocumentHandler_MissingFields(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewUpdateDocumentHandler(client, nil)

	cases := []map[string]any{
		{"ref": "11111111-1111-1111-1111-111111111111"},
		{"document_type": "РеализацияТоваровУслуг"},
	}
	for _, args := range cases {
		raw, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_document", Arguments: raw}}
		if _, err := handler(context.Background(), req); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestUpdateDocumentHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not_found","message":"Документ не найден","field":"ref"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewUpdateDocumentHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":           "00000000-0000-0000-0000-000000000000",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
}

func TestUpdateDocumentTool(t *testing.T) {
	tool := UpdateDocumentTool()
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
	if tool.Name != "update_document" {
		t.Errorf("expected tool name update_document, got %s", tool.Name)
	}
	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
