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

func TestGetDocumentHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document/РеализацияТоваровУслуг/11111111-1111-1111-1111-111111111111" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","type":"РеализацияТоваровУслуг","number":"ЗН-000001","date":"2026-07-02T10:00:00","posted":true,"attributes":{"Организация":"ООО Ромашка"}}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewGetDocumentHandler(client)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":            "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ЗН-000001", "Проведён: true", "ООО Ромашка"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestGetDocumentHandler_ReferenceAttributesAndTabularSections(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ref":"11111111-1111-1111-1111-111111111111",
			"type":"РеализацияТоваровУслуг",
			"number":"ЗН-000001",
			"date":"2026-07-02T10:00:00",
			"posted":false,
			"deletion_mark":true,
			"attributes":{"Контрагент":{"presentation":"ООО Ромашка","ref":"22222222-2222-2222-2222-222222222222"}},
			"tabular_sections":{"Товары":[{"Номенклатура":{"presentation":"Товар 1","ref":"33333333-3333-3333-3333-333333333333"},"Количество":5}]}
		}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewGetDocumentHandler(client)

	args, _ := json.Marshal(map[string]any{
		"document_type": "РеализацияТоваровУслуг",
		"ref":            "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"Пометка удаления: true",
		"ООО Ромашка (ref: 22222222-2222-2222-2222-222222222222)",
		"Товары",
		"Товар 1 (ref: 33333333-3333-3333-3333-333333333333)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestGetDocumentHandler_NotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not_found","message":"Документ не найден: xyz"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewGetDocumentHandler(client)

	args, _ := json.Marshal(map[string]any{"document_type": "РеализацияТоваровУслуг", "ref": "xyz"})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for not_found")
	}
}

func TestGetDocumentTool(t *testing.T) {
	tool := GetDocumentTool()
	if tool.Name != "get_document" {
		t.Errorf("expected get_document, got %s", tool.Name)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint=true")
	}
}
