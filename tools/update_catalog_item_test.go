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

func TestUpdateCatalogItemHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog/update" {
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
		if req.Type != "Контрагенты" {
			t.Errorf("expected type Контрагенты, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ref":"11111111-1111-1111-1111-111111111111",
			"type":"Контрагенты",
			"code":"00001",
			"description":"ООО Ромашка Новая",
			"is_group":false,
			"deletion_mark":false,
			"attributes":{},
			"tabular_sections":{}
		}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewUpdateCatalogItemHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"catalog_type": "Контрагенты",
		"ref":          "11111111-1111-1111-1111-111111111111",
		"attributes":   map[string]any{"Наименование": "ООО Ромашка Новая"},
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result: %v", result.Content)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"изменён", "ООО Ромашка Новая", "00001"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestUpdateCatalogItemHandler_MissingFields(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewUpdateCatalogItemHandler(client, nil)

	cases := []map[string]any{
		{"ref": "11111111-1111-1111-1111-111111111111"},
		{"catalog_type": "Контрагенты"},
	}
	for _, args := range cases {
		raw, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_catalog_item", Arguments: raw}}
		if _, err := handler(context.Background(), req); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestUpdateCatalogItemHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not_found","message":"Элемент не найден","field":"ref"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewUpdateCatalogItemHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"catalog_type": "Контрагенты",
		"ref":          "00000000-0000-0000-0000-000000000000",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "update_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
}

func TestUpdateCatalogItemTool(t *testing.T) {
	tool := UpdateCatalogItemTool()
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
	if tool.Name != "update_catalog_item" {
		t.Errorf("expected tool name update_catalog_item, got %s", tool.Name)
	}
	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
