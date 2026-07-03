package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// documentRefInput is the input shared by post_document and unpost_document.
type documentRefInput struct {
	DocumentType string `json:"document_type"`
	Ref          string `json:"ref"`
}

// PostDocumentTool returns the MCP tool definition for post_document.
// Only registered when write tools are enabled (see server.New).
func PostDocumentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "post_document",
		Title: "Провести документ",
		Description: "Провести (сделать движения по регистрам) ранее созданный документ-черновик. " +
			"Требует ref, полученный от create_document или get_document. " +
			"1С проверит документ по правилам учёта (остатки, закрытые периоды и т.п.) — " +
			"при ошибке вернётся точный текст проверки, которую можно объяснить пользователю или исправить документ и повторить. " +
			"Рекомендуется вызвать get_document перед проведением, чтобы убедиться в правильности данных.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"document_type": {
					"type": "string",
					"description": "Имя вида документа в метаданных 1С, например РеализацияТоваровУслуг"
				},
				"ref": {
					"type": "string",
					"description": "GUID документа, полученный от create_document или get_document"
				}
			},
			"required": ["document_type", "ref"]
		}`),
	}
}

// NewPostDocumentHandler returns a ToolHandler that posts (проводит) an existing document.
func NewPostDocumentHandler(client *onec.Client) mcp.ToolHandler {
	return newDocumentPostingHandler(client, "/document/post", "проведён")
}

// UnpostDocumentTool returns the MCP tool definition for unpost_document.
// Only registered when write tools are enabled (see server.New).
func UnpostDocumentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "unpost_document",
		Title: "Отменить проведение документа",
		Description: "Отменить проведение (распровести) ранее проведённого документа — убирает его движения по регистрам. " +
			"Используй для исправления ошибочно проведённого документа. " +
			"После отмены проведения документ остаётся в базе как черновик — его можно исправить и провести заново через post_document.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"document_type": {
					"type": "string",
					"description": "Имя вида документа в метаданных 1С, например РеализацияТоваровУслуг"
				},
				"ref": {
					"type": "string",
					"description": "GUID документа"
				}
			},
			"required": ["document_type", "ref"]
		}`),
	}
}

// NewUnpostDocumentHandler returns a ToolHandler that unposts (распроводит) an existing document.
func NewUnpostDocumentHandler(client *onec.Client) mcp.ToolHandler {
	return newDocumentPostingHandler(client, "/document/unpost", "отменено проведение")
}

// newDocumentPostingHandler builds the shared post/unpost handler logic — both
// tools send the same {type, ref} body and only differ in the target endpoint
// and success wording.
func newDocumentPostingHandler(client *onec.Client, endpoint, successVerb string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input documentRefInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.DocumentType == "" {
			return nil, fmt.Errorf("document_type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}

		body := onec.DocumentPostRequest{Type: input.DocumentType, Ref: input.Ref}
		var result onec.DocumentWriteResult
		if err := client.Post(ctx, endpoint, body, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("posting document in 1C: %w", err)
		}

		text := fmt.Sprintf("Документ %s: %s (ref=%s, posted=%v, date=%s).",
			input.DocumentType, successVerb, result.Ref, result.Posted, result.Date)
		return textResult(text), nil
	}
}
