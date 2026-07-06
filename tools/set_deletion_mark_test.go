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

func TestSetDeletionMarkHandler_DefaultMarkTrue(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object/deletion-mark" {
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
		var req onec.DeletionMarkRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshaling body: %v", err)
		}
		if req.ObjectKind != "Document" {
			t.Errorf("expected object_kind Document, got %s", req.ObjectKind)
		}
		if !req.Mark {
			t.Error("expected mark defaulted to true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","deletion_mark":true,"posted":false}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewSetDeletionMarkHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"object_kind": "Document",
		"type":        "РеализацияТоваровУслуг",
		"ref":         "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "set_deletion_mark", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result: %v", result.Content)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"11111111-1111-1111-1111-111111111111", "true", "проведения"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestSetDeletionMarkHandler_ExplicitFalse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req onec.DeletionMarkRequest
		json.Unmarshal(body, &req)
		if req.Mark {
			t.Error("expected mark=false to be passed through")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","deletion_mark":false}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewSetDeletionMarkHandler(client, nil)

	mark := false
	args, _ := json.Marshal(map[string]any{
		"object_kind": "Catalog",
		"type":        "Контрагенты",
		"ref":         "11111111-1111-1111-1111-111111111111",
		"mark":        mark,
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "set_deletion_mark", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "false") {
		t.Errorf("expected false in text, got:\n%s", text)
	}
	if strings.Contains(text, "проведения") {
		t.Errorf("catalog response has no posted field, should not mention проведения: %s", text)
	}
}

func TestSetDeletionMarkHandler_MissingFields(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewSetDeletionMarkHandler(client, nil)

	cases := []map[string]any{
		{"type": "Контрагенты", "ref": "11111111-1111-1111-1111-111111111111"},
		{"object_kind": "Catalog", "ref": "11111111-1111-1111-1111-111111111111"},
		{"object_kind": "Catalog", "type": "Контрагенты"},
	}
	for _, args := range cases {
		raw, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "set_deletion_mark", Arguments: raw}}
		if _, err := handler(context.Background(), req); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestSetDeletionMarkHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not_found","message":"Объект не найден","field":"ref"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewSetDeletionMarkHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"object_kind": "Document",
		"type":        "РеализацияТоваровУслуг",
		"ref":         "00000000-0000-0000-0000-000000000000",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "set_deletion_mark", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
}

func TestSetDeletionMarkTool(t *testing.T) {
	tool := SetDeletionMarkTool()
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
	if tool.Name != "set_deletion_mark" {
		t.Errorf("expected tool name set_deletion_mark, got %s", tool.Name)
	}
	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
