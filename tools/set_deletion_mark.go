package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setDeletionMarkInput is the input for the set_deletion_mark tool.
type setDeletionMarkInput struct {
	ObjectKind string `json:"object_kind"`
	Type       string `json:"type"`
	Ref        string `json:"ref"`
	Mark       *bool  `json:"mark,omitempty"`
}

// SetDeletionMarkTool returns the MCP tool definition for set_deletion_mark.
// Only registered when write tools are enabled (see server.New). There is
// deliberately no hard-delete tool — only this reversible mark, matching what
// an interactive 1C user does.
func SetDeletionMarkTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "set_deletion_mark",
		Title: "Пометить на удаление",
		Description: "Установить или снять пометку удаления у документа или элемента справочника. " +
			"Физического удаления нет и не будет — только пометка (как кнопка \"Пометить на удаление\" в интерфейсе 1С). " +
			"Для проведённого документа результат распроведения зависит от бизнес-логики конкретной конфигурации — " +
			"проверяй поле posted в ответе, не предполагай заранее. " +
			"mark по умолчанию true (поставить пометку); передай false, чтобы снять.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"object_kind": {
					"type": "string",
					"enum": ["Document", "Catalog"],
					"description": "Document или Catalog"
				},
				"type": {
					"type": "string",
					"description": "Имя вида документа или справочника в метаданных 1С"
				},
				"ref": {
					"type": "string",
					"description": "GUID объекта"
				},
				"mark": {
					"type": "boolean",
					"description": "true — поставить пометку удаления (по умолчанию), false — снять"
				}
			},
			"required": ["object_kind", "type", "ref"]
		}`),
	}
}

// NewSetDeletionMarkHandler returns a ToolHandler that sets or clears the deletion mark on a document or catalog item.
func NewSetDeletionMarkHandler(client *onec.Client, writeBlacklist []string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input setDeletionMarkInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.ObjectKind == "" {
			return nil, fmt.Errorf("object_kind is required")
		}
		if input.Type == "" {
			return nil, fmt.Errorf("type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}
		if IsWriteBlacklisted(input.Type, writeBlacklist) {
			return blacklistedTypeResult(input.Type), nil
		}
		mark := true
		if input.Mark != nil {
			mark = *input.Mark
		}

		body := onec.DeletionMarkRequest{
			ObjectKind: input.ObjectKind,
			Type:       input.Type,
			Ref:        input.Ref,
			Mark:       mark,
		}
		var result onec.DeletionMarkResult
		if err := client.Post(ctx, "/object/deletion-mark", body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("setting deletion mark in 1C: %w", err)
		}

		text := fmt.Sprintf("Пометка удаления для %s (ref=%s): %v.", input.Type, result.Ref, result.DeletionMark)
		if result.Posted != nil {
			text += fmt.Sprintf(" Текущее состояние проведения: %v.", *result.Posted)
		}
		return textResult(text), nil
	}
}
