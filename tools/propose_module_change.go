package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/feenlace/mcp-1c/dump"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// reviewProposalsDirName is where proposed changes are staged, under the
// --dump directory. It is never read or written by any other tool — there is
// deliberately no "apply" tool that consumes it; a human reviews the .diff,
// merges it into the dump tree by hand, and reloads it into 1C themselves
// (see docs, e.g. via Configurator "Загрузить конфигурацию из файлов" or
// /LoadConfigFromFiles for the specific object).
const reviewProposalsDirName = ".review/proposals"

var unsafeFilenameChars = regexp.MustCompile(`[^\p{L}\p{N}_.-]+`)

// proposeModuleChangeInput is the input for the propose_module_change tool.
type proposeModuleChangeInput struct {
	ModuleID   string `json:"module_id"`
	NewContent string `json:"new_content"`
	Rationale  string `json:"rationale"`
}

// ProposeModuleChangeTool returns the MCP tool definition for propose_module_change.
// Registered whenever a --dump index is configured (same condition as search_code) —
// it only writes review files to disk, never touches the 1C database, so it is not
// gated behind --enable-writes.
func ProposeModuleChangeTool() *mcp.Tool {
	return &mcp.Tool{
		Name:  "propose_module_change",
		Title: "Предложить изменение модуля",
		Description: "Сохранить предложенный вариант кода BSL-модуля как diff для проверки человеком. " +
			"НЕ применяет изменение — ни к выгрузке, ни тем более к базе 1С. Результат сохраняется в " +
			"файл .review/proposals внутри директории --dump; разработчик сам просматривает diff, " +
			"переносит принятые изменения в выгрузку и загружает их в 1С обычным способом " +
			"(Конфигуратор \"Загрузить конфигурацию из файлов\" или LoadConfigFromFiles). " +
			"module_id — тот же идентификатор модуля, что показывает search_code в заголовке результата " +
			"(например Документ.РеализацияТоваровУслуг.МодульОбъекта). " +
			"new_content — ПОЛНЫЙ новый текст модуля (не фрагмент/патч). " +
			"Используй list_proposed_changes чтобы увидеть уже сохранённые предложения.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"module_id": {
					"type": "string",
					"description": "Идентификатор модуля, как в результатах search_code, например Документ.РеализацияТоваровУслуг.МодульОбъекта"
				},
				"new_content": {
					"type": "string",
					"description": "Полный новый текст модуля целиком"
				},
				"rationale": {
					"type": "string",
					"description": "Почему предлагается это изменение"
				}
			},
			"required": ["module_id", "new_content", "rationale"]
		}`),
	}
}

// NewProposeModuleChangeHandler returns a ToolHandler that stages a proposed
// module change as a reviewable diff — it never writes into the dump tree
// itself and never touches 1C.
func NewProposeModuleChangeHandler(index *dump.Index) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input proposeModuleChangeInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("parsing input: %w", err)
		}
		if input.ModuleID == "" {
			return nil, fmt.Errorf("module_id is required")
		}
		if input.NewContent == "" {
			return nil, fmt.Errorf("new_content is required")
		}
		if input.Rationale == "" {
			return nil, fmt.Errorf("rationale is required")
		}

		currentContent, ok := index.GetContent(input.ModuleID)
		if !ok {
			return nil, fmt.Errorf("модуль не найден в индексе: %s (проверьте id через search_code)", input.ModuleID)
		}
		path, _ := index.GetPath(input.ModuleID)
		displayPath := path
		if rel, err := filepath.Rel(index.Dir(), path); err == nil {
			displayPath = filepath.ToSlash(rel)
		}

		diffText := formatUnifiedDiff(displayPath, currentContent, input.NewContent)

		reviewDir := filepath.Join(index.Dir(), filepath.FromSlash(reviewProposalsDirName))
		if err := os.MkdirAll(reviewDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating review directory: %w", err)
		}

		timestamp := time.Now().UTC().Format("20060102T150405Z")
		safeID := unsafeFilenameChars.ReplaceAllString(input.ModuleID, "_")
		baseName := fmt.Sprintf("%s-%s", timestamp, safeID)
		diffPath := filepath.Join(reviewDir, baseName+".diff")
		metaPath := filepath.Join(reviewDir, baseName+".meta.json")

		if err := os.WriteFile(diffPath, []byte(diffText), 0o644); err != nil {
			return nil, fmt.Errorf("writing proposal diff: %w", err)
		}

		meta := proposalMeta{
			ModuleID:  input.ModuleID,
			Path:      displayPath,
			Rationale: input.Rationale,
			Timestamp: timestamp,
		}
		metaJSON, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling proposal metadata: %w", err)
		}
		if err := os.WriteFile(metaPath, metaJSON, 0o644); err != nil {
			return nil, fmt.Errorf("writing proposal metadata: %w", err)
		}

		text := fmt.Sprintf(
			"Предложение сохранено: %s\n"+
				"Модуль: %s (%s)\n"+
				"Обоснование: %s\n\n"+
				"Изменение НЕ применено ни к выгрузке, ни к базе 1С. "+
				"Человек должен просмотреть diff, перенести принятые изменения в выгрузку и "+
				"загрузить их в 1С обычным способом.\n\n%s",
			diffPath, input.ModuleID, displayPath, input.Rationale, diffText,
		)
		return textResult(text), nil
	}
}

// proposalMeta is the JSON sidecar stored next to each .diff file.
type proposalMeta struct {
	ModuleID  string `json:"module_id"`
	Path      string `json:"path"`
	Rationale string `json:"rationale"`
	Timestamp string `json:"timestamp"`
}

// listProposalsDir returns the review-proposals directory for an index,
// shared by propose_module_change and list_proposed_changes.
func listProposalsDir(index *dump.Index) string {
	return filepath.Join(index.Dir(), filepath.FromSlash(reviewProposalsDirName))
}
