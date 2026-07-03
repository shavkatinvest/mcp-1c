package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// getDocumentInput is the input for the get_document tool.
type getDocumentInput struct {
	DocumentType string `json:"document_type"`
	Ref          string `json:"ref"`
}

// GetDocumentTool returns the MCP tool definition for get_document.
// Unlike create_document/post_document/unpost_document, this tool is read-only
// and is registered regardless of whether write tools are enabled.
func GetDocumentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "get_document",
		Title: "Прочитать состояние документа",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Description: "Прочитать текущее состояние документа по ref: проведён/черновик, номер, дата, значения реквизитов шапки. " +
			"Используй перед post_document, чтобы убедиться в правильности данных документа.",
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

// NewGetDocumentHandler returns a ToolHandler that reads current document state from 1C.
func NewGetDocumentHandler(client *onec.Client) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input getDocumentInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.DocumentType == "" {
			return nil, fmt.Errorf("document_type is required")
		}
		if input.Ref == "" {
			return nil, fmt.Errorf("ref is required")
		}

		endpoint := fmt.Sprintf("/document/%s/%s", input.DocumentType, input.Ref)
		var result onec.DocumentGetResult
		if err := client.Get(ctx, endpoint, &result); err != nil {
			if errResult, ok := writeErrorResult(err); ok {
				return errResult, nil
			}
			return nil, fmt.Errorf("reading document from 1C: %w", err)
		}

		return textResult(formatDocument(&result)), nil
	}
}

func formatDocument(r *onec.DocumentGetResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s №%s от %s\n\n", r.Type, r.Number, r.Date)
	fmt.Fprintf(&b, "- ref: %s\n", r.Ref)
	fmt.Fprintf(&b, "- Проведён: %v\n\n", r.Posted)

	if len(r.Attributes) == 0 {
		b.WriteString("Реквизиты отсутствуют.\n")
		return b.String()
	}

	b.WriteString("### Реквизиты\n\n")
	for name, value := range r.Attributes {
		fmt.Fprintf(&b, "- %s: %v\n", name, value)
	}
	return b.String()
}
