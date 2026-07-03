package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createDocumentInput is the input for the create_document tool.
type createDocumentInput struct {
	DocumentType    string                      `json:"document_type"`
	Attributes      map[string]any              `json:"attributes,omitempty"`
	TabularSections map[string][]map[string]any `json:"tabular_sections,omitempty"`
}

// CreateDocumentTool returns the MCP tool definition for create_document.
// Only registered when write tools are enabled (see server.New).
func CreateDocumentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "create_document",
		Title: "Создать документ (черновик)",
		Description: "Создать новый документ 1С — реализацию, поступление, платёжное поручение и т.п. " +
			"Создаёт ТОЛЬКО черновик (не проведённый документ) — эта функция не может провести документ, " +
			"даже если это попросить: параметра для проведения здесь нет. " +
			"Чтобы провести документ, после создания вызови post_document с полученным ref. " +
			"Перед проведением рекомендуется проверить содержимое через get_document. " +
			"Имена реквизитов и табличных частей бери из get_object_structure. " +
			"Значения ссылочных реквизитов (Контрагент, Организация, Номенклатура и т.п.) передавай как GUID — " +
			"получи его через execute_query (поле Ссылка возвращает представление; используй УникальныйИдентификатор(Ссылка) в запросе).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"document_type": {
					"type": "string",
					"description": "Имя вида документа в метаданных 1С, например РеализацияТоваровУслуг"
				},
				"attributes": {
					"type": "object",
					"description": "Значения реквизитов шапки документа, ключ — имя реквизита. Пример: {\"Дата\": \"2026-07-02T10:00:00\", \"Организация\": \"<GUID>\", \"Контрагент\": \"<GUID>\"}"
				},
				"tabular_sections": {
					"type": "object",
					"description": "Табличные части документа. Ключ — имя табличной части, значение — массив строк (объектов с именами реквизитов). Пример: {\"Товары\": [{\"Номенклатура\": \"<GUID>\", \"Количество\": 10, \"Цена\": 1500}]}"
				}
			},
			"required": ["document_type"]
		}`),
	}
}

// NewCreateDocumentHandler returns a ToolHandler that creates an unposted document draft in 1C.
func NewCreateDocumentHandler(client *onec.Client) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input createDocumentInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.DocumentType == "" {
			return nil, fmt.Errorf("document_type is required")
		}

		body := onec.CreateDocumentRequest{
			Type:            input.DocumentType,
			Attributes:      input.Attributes,
			TabularSections: input.TabularSections,
		}
		var result onec.DocumentWriteResult
		if err := client.Post(ctx, "/document", body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("creating document in 1C: %w", err)
		}

		text := fmt.Sprintf(
			"Создан черновик документа %s №%s от %s (ref=%s, posted=%v).\n"+
				"Документ НЕ проведён. Чтобы провести — вызови post_document с document_type=%q, ref=%q.\n"+
				"Перед проведением рекомендуется проверить данные через get_document.",
			input.DocumentType, result.Number, result.Date, result.Ref, result.Posted,
			input.DocumentType, result.Ref,
		)
		return textResult(text), nil
	}
}

// writeErrorResult converts a StatusError carrying a structured 1C write-endpoint
// error ({"error": code, "message": text, "field": ...}) into an MCP tool-level
// error result (IsError: true) so the LLM sees the exact reason and can
// self-correct, instead of a generic Go error. Returns ok=false if err is not
// a StatusError with a parseable structured body — callers should fall back to
// returning a plain Go error in that case.
func writeErrorResult(err error) (*mcp.CallToolResult, bool) {
	statusErr, ok := err.(*onec.StatusError)
	if !ok {
		return nil, false
	}
	apiErr, ok := statusErr.APIError()
	if !ok {
		return nil, false
	}

	text := fmt.Sprintf("Ошибка 1С [%s]: %s", apiErr.Code, apiErr.Message)
	if apiErr.Field != "" {
		text += fmt.Sprintf(" (поле: %s)", apiErr.Field)
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, true
}
