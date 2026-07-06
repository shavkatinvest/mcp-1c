package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// getCatalogItemInput is the input for the get_catalog_item tool.
type getCatalogItemInput struct {
	CatalogType string `json:"catalog_type"`
	Ref         string `json:"ref"`
}

// GetCatalogItemTool returns the MCP tool definition for get_catalog_item.
// Read-only, always registered regardless of write mode.
func GetCatalogItemTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "get_catalog_item",
		Title: "Прочитать элемент справочника",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Description: "Прочитать текущее состояние элемента справочника по ref: код, наименование, реквизиты, родитель, владелец, табличные части. " +
			"Используй перед update_catalog_item, чтобы проверить текущие значения.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"catalog_type": {
					"type": "string",
					"description": "Имя справочника в метаданных 1С, например Контрагенты"
				},
				"ref": {
					"type": "string",
					"description": "GUID элемента справочника"
				}
			},
			"required": ["catalog_type", "ref"]
		}`),
	}
}

// NewGetCatalogItemHandler returns a ToolHandler that reads current catalog item state from 1C.
func NewGetCatalogItemHandler(client *onec.Client) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input getCatalogItemInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.CatalogType == "" {
			return nil, fmt.Errorf("catalog_type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}

		endpoint := fmt.Sprintf("/catalog/%s/%s", input.CatalogType, input.Ref)
		var result onec.CatalogGetResult
		if err := client.Get(ctx, endpoint, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("reading catalog item from 1C: %w", err)
		}

		return textResult(formatCatalogItem(&result)), nil
	}
}

func formatCatalogItem(r *onec.CatalogGetResult) string {
	var b strings.Builder
	title := r.Description
	if title == "" {
		title = r.Code
	}
	fmt.Fprintf(&b, "## %s: %s\n\n", r.Type, title)
	fmt.Fprintf(&b, "- ref: %s\n", r.Ref)
	if r.Code != "" {
		fmt.Fprintf(&b, "- Код: %s\n", r.Code)
	}
	fmt.Fprintf(&b, "- Группа: %v\n", r.IsGroup)
	fmt.Fprintf(&b, "- Пометка удаления: %v\n", r.DeletionMark)
	if r.Parent != "" {
		fmt.Fprintf(&b, "- Родитель: %s\n", r.Parent)
	}
	if r.Owner != "" {
		fmt.Fprintf(&b, "- Владелец: %s\n", r.Owner)
	}
	b.WriteString("\n")

	if len(r.Attributes) == 0 {
		b.WriteString("Реквизиты отсутствуют.\n")
	} else {
		b.WriteString("### Реквизиты\n\n")
		for name, value := range r.Attributes {
			fmt.Fprintf(&b, "- %s: %s\n", name, formatAttributeValue(value))
		}
		b.WriteString("\n")
	}

	b.WriteString(formatTabularSections(r.TabularSections))
	return b.String()
}
