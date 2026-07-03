package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/feenlace/mcp-1c/dump"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListProposedChangesTool returns the MCP tool definition for list_proposed_changes.
func ListProposedChangesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "list_proposed_changes",
		Title: "Список предложенных изменений",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Description: "Показать предложения изменений модулей, сохранённые через propose_module_change " +
			"и ещё не удалённые вручную (например после переноса в выгрузку). Полезно чтобы продолжить " +
			"работу над предложениями в новой сессии.",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
	}
}

// NewListProposedChangesHandler returns a ToolHandler that lists pending proposals.
func NewListProposedChangesHandler(index *dump.Index) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dir := listProposalsDir(index)

		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return textResult("Предложений нет (.review/proposals ещё не создана)."), nil
			}
			return nil, fmt.Errorf("reading proposals directory: %w", err)
		}

		var metaFiles []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".meta.json") {
				metaFiles = append(metaFiles, e.Name())
			}
		}
		sort.Strings(metaFiles)

		if len(metaFiles) == 0 {
			return textResult("Предложений нет."), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Предложенные изменения (%d)\n\n", len(metaFiles))
		for _, name := range metaFiles {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			var meta proposalMeta
			if err := json.Unmarshal(data, &meta); err != nil {
				continue
			}
			diffName := strings.TrimSuffix(name, ".meta.json") + ".diff"
			fmt.Fprintf(&b, "- **%s** (%s)\n  - Файл: `%s`\n  - Обоснование: %s\n",
				meta.ModuleID, meta.Timestamp, filepath.Join(dir, diffName), meta.Rationale)
		}

		return textResult(b.String()), nil
	}
}
