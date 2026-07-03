package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListProposedChangesHandler_Empty(t *testing.T) {
	dir := t.TempDir()
	mkBSL(t, dir, "Documents/Тест/Ext/ObjectModule.bsl", "Процедура Тест()\nКонецПроцедуры\n")
	index := newTestIndex(t, dir)

	handler := NewListProposedChangesHandler(index)
	result, err := callTool(t, handler, "list_proposed_changes", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Предложений нет") {
		t.Errorf("expected empty-state message, got:\n%s", text)
	}
}

func TestListProposedChangesHandler_AfterPropose(t *testing.T) {
	dir := t.TempDir()
	mkBSL(t, dir, "Documents/РеализацияТоваровУслуг/Ext/ObjectModule.bsl",
		"Процедура ОбработкаПроведения(Отказ, РежимПроведения)\nКонецПроцедуры\n")
	index := newTestIndex(t, dir)

	proposeHandler := NewProposeModuleChangeHandler(index)
	if _, err := callTool(t, proposeHandler, "propose_module_change", map[string]any{
		"module_id":   "Документ.РеализацияТоваровУслуг.МодульОбъекта",
		"new_content": "Процедура ОбработкаПроведения(Отказ, РежимПроведения)\n\t// изменено\nКонецПроцедуры\n",
		"rationale":   "test rationale",
	}); err != nil {
		t.Fatalf("propose_module_change: %v", err)
	}

	listHandler := NewListProposedChangesHandler(index)
	result, err := callTool(t, listHandler, "list_proposed_changes", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{
		"Документ.РеализацияТоваровУслуг.МодульОбъекта",
		"test rationale",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in list, got:\n%s", want, text)
		}
	}
}

func TestListProposedChangesTool(t *testing.T) {
	tool := ListProposedChangesTool()
	if tool.Name != "list_proposed_changes" {
		t.Errorf("expected list_proposed_changes, got %s", tool.Name)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint=true")
	}
}
