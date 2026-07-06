package tools

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult wraps a text string into an MCP tool result.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// objectInput is the common input for tools that operate on a specific metadata object.
type objectInput struct {
	ObjectType string `json:"object_type"`
	ObjectName string `json:"object_name"`
}

// queryLimitInput is the common input for tools that accept a query string with an optional limit.
type queryLimitInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// searchCodeInput is the input for the search_code tool.
type searchCodeInput struct {
	Query    string `json:"query"`
	Limit    int    `json:"limit"`
	Category string `json:"category"`
	Module   string `json:"module"`
	Mode     string `json:"mode"`
}

// clampLimit normalises a user-supplied limit to [defaultVal, maxVal].
func clampLimit(value, defaultVal, maxVal int) int {
	if value <= 0 {
		return defaultVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// formatAttributeValue renders a single attribute/tabular-cell value for
// display. Reference-typed values arrive from 1C as {"presentation": "...",
// "ref": "<GUID>"} (see ЗначениеАтрибутаКJSON on the 1C side) so update/find
// calls can reuse the ref without a separate lookup; this renders them as
// "presentation (ref: guid)" instead of Go's default map syntax. Any other
// JSON value (string/number/bool) is rendered with %v as before.
func formatAttributeValue(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	presentation, hasPresentation := m["presentation"]
	ref, hasRef := m["ref"]
	if !hasPresentation || !hasRef {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("%v (ref: %v)", presentation, ref)
}

// formatTabularSections renders the {"ИмяТЧ": [{...}, ...]} shape shared by
// document and catalog GET/update responses as markdown tables.
func formatTabularSections(sections map[string][]map[string]any) string {
	if len(sections) == 0 {
		return ""
	}
	var b strings.Builder
	for name, rows := range sections {
		fmt.Fprintf(&b, "### %s\n\n", name)
		if len(rows) == 0 {
			b.WriteString("(пусто)\n\n")
			continue
		}
		for i, row := range rows {
			fmt.Fprintf(&b, "**Строка %d**\n", i+1)
			for field, value := range row {
				fmt.Fprintf(&b, "- %s: %s\n", field, formatAttributeValue(value))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
