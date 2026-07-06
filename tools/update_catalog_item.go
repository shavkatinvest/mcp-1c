package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// updateCatalogItemInput is the input for the update_catalog_item tool.
type updateCatalogItemInput struct {
	CatalogType     string                      `json:"catalog_type"`
	Ref             string                      `json:"ref"`
	Attributes      map[string]any              `json:"attributes,omitempty"`
	TabularSections map[string][]map[string]any `json:"tabular_sections,omitempty"`
}

// UpdateCatalogItemTool returns the MCP tool definition for update_catalog_item.
// Only registered when write tools are enabled (see server.New).
func UpdateCatalogItemTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "update_catalog_item",
		Title: "Изменить элемент справочника",
		Description: "Изменить реквизиты и/или табличные части существующего элемента справочника по ref. " +
			"Стандартные реквизиты (Наименование, Код, Родитель, Владелец) передавай прямо в attributes. " +
			"ВАЖНО: указанная табличная часть заменяется ЦЕЛИКОМ — отправляй полный набор строк, а не только изменённые.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"catalog_type": {
					"type": "string",
					"description": "Имя справочника в метаданных 1С"
				},
				"ref": {
					"type": "string",
					"description": "GUID элемента справочника"
				},
				"attributes": {
					"type": "object",
					"description": "Изменяемые реквизиты (только те, что нужно изменить)"
				},
				"tabular_sections": {
					"type": "object",
					"description": "Табличные части для полной замены."
				}
			},
			"required": ["catalog_type", "ref"]
		}`),
	}
}

// NewUpdateCatalogItemHandler returns a ToolHandler that updates an existing catalog item's attributes/tabular sections.
func NewUpdateCatalogItemHandler(client *onec.Client, writeBlacklist []string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input updateCatalogItemInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.CatalogType == "" {
			return nil, fmt.Errorf("catalog_type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}
		if IsWriteBlacklisted(input.CatalogType, writeBlacklist) {
			return blacklistedTypeResult(input.CatalogType), nil
		}

		body := onec.UpdateObjectRequest{
			Type:            input.CatalogType,
			Ref:             input.Ref,
			Attributes:      input.Attributes,
			TabularSections: input.TabularSections,
		}
		var result onec.CatalogGetResult
		if err := client.Post(ctx, "/catalog/update", body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("updating catalog item in 1C: %w", err)
		}

		return textResult("Элемент справочника изменён.\n\n" + formatCatalogItem(&result)), nil
	}
}
