package tools

import (
	"fmt"
	"path"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultWriteTypeBlacklist lists document/catalog type-name glob patterns that
// write tools refuse to touch regardless of the underlying 1C user's rights.
// Payment documents integrated with online banking (dibank_*) are excluded by
// default because a wrong or malicious AI-issued edit there can move real
// money; this only gates the MCP write path — human/Configurator access to
// these documents is unaffected. See docs/WRITE-TOOLS.md.
var DefaultWriteTypeBlacklist = []string{"dibank_*", "дибанк_*"}

// IsWriteBlacklisted reports whether typeName matches any of the glob patterns
// in blacklist. Matching is case-insensitive; patterns use path.Match syntax
// (*, ?, [...]). A nil or empty blacklist matches nothing.
func IsWriteBlacklisted(typeName string, blacklist []string) bool {
	lower := strings.ToLower(typeName)
	for _, pattern := range blacklist {
		if matched, _ := path.Match(strings.ToLower(pattern), lower); matched {
			return true
		}
	}
	return false
}

// blacklistedTypeResult builds the IsError tool result returned when a write
// tool is called against a blacklisted type, instead of making the 1C request.
func blacklistedTypeResult(typeName string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"Тип %q заблокирован политикой сервера для операций записи (платёжные документы). "+
				"Это ограничение уровня MCP-сервера и не связано с правами 1С-пользователя.",
			typeName,
		)}},
	}
}
