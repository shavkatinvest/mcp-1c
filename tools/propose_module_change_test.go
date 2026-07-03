package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/feenlace/mcp-1c/dump"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestIndex(t *testing.T, dir string) *dump.Index {
	t.Helper()
	index, err := dump.NewIndex(dir, "", false)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	t.Cleanup(func() { index.Close() })
	waitReady(t, index, 30*time.Second)
	return index
}

func callTool(t *testing.T, handler mcp.ToolHandler, name string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	raw, _ := json.Marshal(args)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: name, Arguments: raw},
	}
	return handler(context.Background(), req)
}

func TestProposeModuleChangeHandler(t *testing.T) {
	dir := t.TempDir()
	mkBSL(t, dir, "Documents/РеализацияТоваровУслуг/Ext/ObjectModule.bsl",
		"Процедура ОбработкаПроведения(Отказ, РежимПроведения)\n\t// старый код\nКонецПроцедуры\n")
	index := newTestIndex(t, dir)

	handler := NewProposeModuleChangeHandler(index)
	result, err := callTool(t, handler, "propose_module_change", map[string]any{
		"module_id":   "Документ.РеализацияТоваровУслуг.МодульОбъекта",
		"new_content": "Процедура ОбработкаПроведения(Отказ, РежимПроведения)\n\t// новый код\nКонецПроцедуры\n",
		"rationale":   "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "НЕ применено") {
		t.Errorf("expected 'not applied' notice, got:\n%s", text)
	}
	if !strings.Contains(text, "новый код") {
		t.Errorf("expected diff to show new content, got:\n%s", text)
	}

	// The original file on disk must be untouched.
	original, err := os.ReadFile(filepath.Join(dir, "Documents/РеализацияТоваровУслуг/Ext/ObjectModule.bsl"))
	if err != nil {
		t.Fatalf("reading original file: %v", err)
	}
	if !strings.Contains(string(original), "старый код") {
		t.Error("expected original file to remain unchanged after propose_module_change")
	}

	// The proposal must be staged under .review/proposals.
	entries, err := os.ReadDir(filepath.Join(dir, ".review", "proposals"))
	if err != nil {
		t.Fatalf("reading proposals dir: %v", err)
	}
	if len(entries) != 2 { // .diff + .meta.json
		t.Errorf("expected 2 files in proposals dir (diff + meta), got %d", len(entries))
	}
}

func TestProposeModuleChangeHandler_ModuleNotFound(t *testing.T) {
	dir := t.TempDir()
	mkBSL(t, dir, "Documents/Тест/Ext/ObjectModule.bsl", "Процедура Тест()\nКонецПроцедуры\n")
	index := newTestIndex(t, dir)

	handler := NewProposeModuleChangeHandler(index)
	_, err := callTool(t, handler, "propose_module_change", map[string]any{
		"module_id":   "Документ.НеСуществует.МодульОбъекта",
		"new_content": "что-то",
		"rationale":   "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown module_id")
	}
}

func TestProposeModuleChangeHandler_MissingFields(t *testing.T) {
	dir := t.TempDir()
	mkBSL(t, dir, "Documents/Тест/Ext/ObjectModule.bsl", "Процедура Тест()\nКонецПроцедуры\n")
	index := newTestIndex(t, dir)
	handler := NewProposeModuleChangeHandler(index)

	cases := []map[string]any{
		{"new_content": "x", "rationale": "y"},
		{"module_id": "x", "rationale": "y"},
		{"module_id": "x", "new_content": "y"},
	}
	for _, args := range cases {
		if _, err := callTool(t, handler, "propose_module_change", args); err == nil {
			t.Errorf("expected error for incomplete args: %v", args)
		}
	}
}

func TestProposeModuleChangeTool(t *testing.T) {
	tool := ProposeModuleChangeTool()
	if tool.Name != "propose_module_change" {
		t.Errorf("expected propose_module_change, got %s", tool.Name)
	}
}
