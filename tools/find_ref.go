package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultFindLimit = 20
	maxFindLimit     = 50
)

// findRefInput is the input for the find_ref tool.
type findRefInput struct {
	ObjectKind    string `json:"object_kind"`
	Name          string `json:"name"`
	Query         string `json:"query"`
	Limit         int    `json:"limit,omitempty"`
	IncludeMarked bool   `json:"include_marked,omitempty"`
}

// FindRefTool returns the MCP tool definition for find_ref. Always registered
// (read-only) regardless of write mode.
func FindRefTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "find_ref",
		Title: "Найти GUID по названию",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Description: "Найти GUID объекта (справочника или документа) по человекочитаемому названию, коду или номеру. " +
			"ВСЕГДА используй этот инструмент, чтобы получить ref для ссылочных реквизитов (Контрагент, Организация, Номенклатура, Склад и т.п.) " +
			"перед create_document/create_catalog_item/update_document/update_catalog_item — никогда не изобретай GUID. " +
			"В текущей версии поддерживаются только object_kind=Catalog и object_kind=Document.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"object_kind": {
					"type": "string",
					"enum": ["Catalog", "Document"],
					"description": "Вид метаданных: Catalog (справочник) или Document (документ)"
				},
				"name": {
					"type": "string",
					"description": "Имя вида метаданных, например Контрагенты или РеализацияТоваровУслуг"
				},
				"query": {
					"type": "string",
					"description": "Текст поиска — часть наименования/кода (для справочников) или номера (для документов)"
				},
				"limit": {
					"type": "integer",
					"description": "Максимальное количество результатов (по умолчанию 20, максимум 50)"
				},
				"include_marked": {
					"type": "boolean",
					"description": "Включать объекты, помеченные на удаление (по умолчанию false)"
				}
			},
			"required": ["object_kind", "name", "query"]
		}`),
	}
}

// NewFindRefHandler returns a ToolHandler that resolves a human-readable name to a GUID.
func NewFindRefHandler(client *onec.Client) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input findRefInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.ObjectKind == "" {
			return nil, fmt.Errorf("object_kind is required")
		}
		if input.Name == "" {
			return nil, fmt.Errorf("name is required")
		}

		limit := clampLimit(input.Limit, defaultFindLimit, maxFindLimit)

		q := url.Values{}
		q.Set("type", input.ObjectKind)
		q.Set("name", input.Name)
		q.Set("query", input.Query)
		q.Set("limit", strconv.Itoa(limit))
		if input.IncludeMarked {
			q.Set("include_marked", "true")
		}

		var result onec.FindRefResult
		if err := client.Get(ctx, "/find?"+q.Encode(), &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("finding reference in 1C: %w", err)
		}

		return textResult(formatFindRefResult(&result)), nil
	}
}

func formatFindRefResult(r *onec.FindRefResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Найдено (%d)\n\n", r.Total)
	if len(r.Items) == 0 {
		b.WriteString("Ничего не найдено.\n")
		return b.String()
	}
	for _, item := range r.Items {
		fmt.Fprintf(&b, "- **%s** (ref: %s", item.Presentation, item.Ref)
		if item.Code != "" {
			fmt.Fprintf(&b, ", код: %s", item.Code)
		}
		if item.IsGroup {
			b.WriteString(", группа")
		}
		if item.DeletionMark {
			b.WriteString(", ПОМЕЧЕН НА УДАЛЕНИЕ")
		}
		if item.Posted != nil {
			fmt.Fprintf(&b, ", проведён: %v", *item.Posted)
		}
		b.WriteString(")\n")
	}
	if r.Truncated {
		b.WriteString("\n> Результат усечён. Уточните запрос или увеличьте limit.\n")
	}
	return b.String()
}
