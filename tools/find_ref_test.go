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

func TestFindRefHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/find" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("type") != "Catalog" || q.Get("name") != "Контрагенты" || q.Get("query") != "Ромашка" {
			t.Errorf("unexpected query params: %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"items": [{"ref":"11111111-1111-1111-1111-111111111111","presentation":"ООО Ромашка","code":"00001","type_name":"СправочникСсылка.Контрагенты","deletion_mark":false,"is_group":false}],
			"total": 1,
			"truncated": false
		}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewFindRefHandler(client)

	args, _ := json.Marshal(map[string]any{
		"object_kind": "Catalog",
		"name":        "Контрагенты",
		"query":       "Ромашка",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "find_ref", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"ООО Ромашка", "11111111-1111-1111-1111-111111111111", "00001"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in text, got:\n%s", want, text)
		}
	}
}

func TestFindRefHandler_Empty(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items": [], "total": 0, "truncated": false}`))
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewFindRefHandler(client)

	args, _ := json.Marshal(map[string]any{
		"object_kind": "Catalog",
		"name":        "Контрагенты",
		"query":       "НесуществующийКонтрагент",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "find_ref", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Ничего не найдено") {
		t.Errorf("expected empty-result message, got:\n%s", text)
	}
}

func TestFindRefHandler_MissingFields(t *testing.T) {
	client := onec.NewClient("http://unused.invalid", "", "")
	handler := NewFindRefHandler(client)

	cases := []map[string]any{
		{"name": "Контрагенты", "query": "x"},
		{"object_kind": "Catalog", "query": "x"},
	}
	for _, args := range cases {
		raw, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "find_ref", Arguments: raw}}
		if _, err := handler(context.Background(), req); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestFindRefTool(t *testing.T) {
	tool := FindRefTool()
	if tool.Name != "find_ref" {
		t.Errorf("expected find_ref, got %s", tool.Name)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint=true")
	}
}
