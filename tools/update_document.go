package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// updateDocumentInput is the input for the update_document tool.
type updateDocumentInput struct {
	DocumentType    string                      `json:"document_type"`
	Ref             string                      `json:"ref"`
	Attributes      map[string]any              `json:"attributes,omitempty"`
	TabularSections map[string][]map[string]any `json:"tabular_sections,omitempty"`
}

// UpdateDocumentTool returns the MCP tool definition for update_document.
// Only registered when write tools are enabled (see server.New).
func UpdateDocumentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "update_document",
		Title: "Изменить документ",
		Description: "Изменить реквизиты и/или табличные части существующего документа по ref. " +
			"ВАЖНО: указанная табличная часть заменяется ЦЕЛИКОМ — отправляй полный набор строк, а не только изменённые. " +
			"Табличные части, не указанные в запросе, не затрагиваются. " +
			"Изменение НИКОГДА не проводит и не распроводит документ — если документ уже проведён, после изменения вызови " +
			"post_document заново, чтобы перепровести с новыми данными.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"document_type": {
					"type": "string",
					"description": "Имя вида документа в метаданных 1С"
				},
				"ref": {
					"type": "string",
					"description": "GUID документа"
				},
				"attributes": {
					"type": "object",
					"description": "Изменяемые реквизиты шапки (только те, что нужно изменить)"
				},
				"tabular_sections": {
					"type": "object",
					"description": "Табличные части для полной замены. Ключ — имя табличной части, значение — ПОЛНЫЙ новый список строк."
				}
			},
			"required": ["document_type", "ref"]
		}`),
	}
}

// NewUpdateDocumentHandler returns a ToolHandler that updates an existing document's attributes/tabular sections.
func NewUpdateDocumentHandler(client *onec.Client, writeBlacklist []string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input updateDocumentInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.DocumentType == "" {
			return nil, fmt.Errorf("document_type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}
		if IsWriteBlacklisted(input.DocumentType, writeBlacklist) {
			return blacklistedTypeResult(input.DocumentType), nil
		}

		body := onec.UpdateObjectRequest{
			Type:            input.DocumentType,
			Ref:             input.Ref,
			Attributes:      input.Attributes,
			TabularSections: input.TabularSections,
		}
		var result onec.DocumentGetResult
		if err := client.Post(ctx, "/document/update", body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("updating document in 1C: %w", err)
		}

		return textResult("Документ изменён.\n\n" + formatDocument(&result)), nil
	}
}
