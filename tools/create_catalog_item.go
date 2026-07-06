package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createCatalogItemInput is the input for the create_catalog_item tool.
type createCatalogItemInput struct {
	CatalogType     string                      `json:"catalog_type"`
	IsGroup         bool                        `json:"is_group,omitempty"`
	OwnerType       string                      `json:"owner_type,omitempty"`
	Attributes      map[string]any              `json:"attributes,omitempty"`
	TabularSections map[string][]map[string]any `json:"tabular_sections,omitempty"`
}

// CreateCatalogItemTool returns the MCP tool definition for create_catalog_item.
// Only registered when write tools are enabled (see server.New).
func CreateCatalogItemTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "create_catalog_item",
		Title: "Создать элемент справочника",
		Description: "Создать новый элемент (или группу, если is_group=true) справочника — контрагента, номенклатуру, склад и т.п. " +
			"Наименование/Код/Родитель/Владелец передавай прямо в attributes как обычные реквизиты — сервер сам знает, что это стандартные поля. " +
			"Значения ссылочных реквизитов (Родитель, Владелец и обычные ссылочные реквизиты) передавай как GUID, полученный через find_ref. " +
			"owner_type нужен только если у справочника несколько видов владельцев и нужно явно указать, какой.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"catalog_type": {
					"type": "string",
					"description": "Имя справочника в метаданных 1С, например Контрагенты"
				},
				"is_group": {
					"type": "boolean",
					"description": "Создать группу вместо элемента (только для иерархических справочников)"
				},
				"owner_type": {
					"type": "string",
					"description": "Имя справочника-владельца (только если у справочника несколько видов владельцев)"
				},
				"attributes": {
					"type": "object",
					"description": "Значения реквизитов, включая стандартные (Наименование, Код, Родитель, Владелец) и обычные. Пример: {\"Наименование\": \"ООО Ромашка\", \"ИНН\": \"301234567\"}"
				},
				"tabular_sections": {
					"type": "object",
					"description": "Табличные части справочника. Ключ — имя табличной части, значение — массив строк."
				}
			},
			"required": ["catalog_type"]
		}`),
	}
}

// NewCreateCatalogItemHandler returns a ToolHandler that creates a catalog item or group in 1C.
func NewCreateCatalogItemHandler(client *onec.Client, writeBlacklist []string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input createCatalogItemInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.CatalogType == "" {
			return nil, fmt.Errorf("catalog_type is required")
		}
		if IsWriteBlacklisted(input.CatalogType, writeBlacklist) {
			return blacklistedTypeResult(input.CatalogType), nil
		}

		body := onec.CreateCatalogItemRequest{
			Type:            input.CatalogType,
			IsGroup:         input.IsGroup,
			OwnerType:       input.OwnerType,
			Attributes:      input.Attributes,
			TabularSections: input.TabularSections,
		}
		var result onec.CatalogWriteResult
		if err := client.Post(ctx, "/catalog", body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("creating catalog item in 1C: %w", err)
		}

		text := fmt.Sprintf(
			"Создан элемент справочника %s: %q (ref=%s, код=%s, группа=%v).",
			input.CatalogType, result.Description, result.Ref, result.Code, result.IsGroup,
		)
		return textResult(text), nil
	}
}
