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

func TestCreateCatalogItemHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
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
		var req onec.CreateCatalogItemRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshaling body: %v", err)
		}
		if req.Type != "Контрагенты" {
			t.Errorf("expected type Контрагенты, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ref":"11111111-1111-1111-1111-111111111111","code":"00001","description":"ООО Ромашка","is_group":false}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateCatalogItemHandler(client, nil)

	args, _ := json.Marshal(map[string]any{
		"catalog_type": "Контрагенты",
		"attributes":   map[string]any{"Наименование": "ООО Ромашка", "ИНН": "301234567"},
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "create_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result: %v", result.Content)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"11111111-1111-1111-1111-111111111111", "ООО Ромашка", "00001"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestCreateCatalogItemHandler_MissingType(t *testing.T) {
	httpCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled = true
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateCatalogItemHandler(client, nil)

	args, _ := json.Marshal(map[string]any{})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "create_catalog_item", Arguments: args}}

	if _, err := handler(context.Background(), req); err == nil {
		t.Fatal("expected error for missing catalog_type")
	}
	if httpCalled {
		t.Error("HTTP call should not have been made without catalog_type")
	}
}

func TestCreateCatalogItemHandler_StructuredError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"unknown_catalog_type","message":"Справочник 'Foo' не найден","field":"catalog_type"}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateCatalogItemHandler(client, nil)

	args, _ := json.Marshal(map[string]any{"catalog_type": "Foo"})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "create_catalog_item", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected structured tool error, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for structured 1C error")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"unknown_catalog_type", "Справочник 'Foo' не найден", "catalog_type"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in error text, got:\n%s", want, text)
		}
	}
}

func TestCreateCatalogItemTool(t *testing.T) {
	tool := CreateCatalogItemTool()
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
	if tool.Name != "create_catalog_item" {
		t.Errorf("expected tool name create_catalog_item, got %s", tool.Name)
	}
	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
