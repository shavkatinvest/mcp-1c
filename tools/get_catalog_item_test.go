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

func TestGetCatalogItemHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog/Контрагенты/11111111-1111-1111-1111-111111111111" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ref":"11111111-1111-1111-1111-111111111111",
			"type":"Контрагенты",
			"code":"00001",
			"description":"ООО Ромашка",
			"is_group":false,
			"deletion_mark":false,
			"attributes":{"ИНН":"301234567"},
			"tabular_sections":{}
		}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewGetCatalogItemHandler(client)

	args, _ := json.Marshal(map[string]any{
		"catalog_type": "Контрагенты",
		"ref":          "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ООО Ромашка", "00001", "ИНН", "301234567"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestGetCatalogItemHandler_MissingFields(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewGetCatalogItemHandler(client)

	cases := []map[string]any{
		{"ref": "11111111-1111-1111-1111-111111111111"},
		{"catalog_type": "Контрагенты"},
	}
	for _, args := range cases {
		raw, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_catalog_item", Arguments: raw}}
		if _, err := handler(context.Background(), req); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestGetCatalogItemHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not_found","message":"Элемент не найден","field":"ref"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewGetCatalogItemHandler(client)

	args, _ := json.Marshal(map[string]any{
		"catalog_type": "Контрагенты",
		"ref":          "00000000-0000-0000-0000-000000000000",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
}

func TestGetCatalogItemTool(t *testing.T) {
	tool := GetCatalogItemTool()
	if tool.Name != "get_catalog_item" {
		t.Errorf("expected get_catalog_item, got %s", tool.Name)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint=true")
	}
}
