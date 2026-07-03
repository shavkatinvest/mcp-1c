package tools

import (
	"fmt"
	"strings"
)

// maxDiffLines caps the LCS diff computation (O(n*m) time and space) to avoid
// pathological memory use on unusually large modules. Above this size, the
// proposal still gets recorded but with the full new content instead of a
// line diff — the reviewer sees the change either way.
const maxDiffLines = 3000

type diffLineKind int

const (
	diffSame diffLineKind = iota
	diffRemoved
	diffAdded
)

type diffLine struct {
	kind diffLineKind
	text string
}

// diffLines computes a line-level diff between old and new via a classic
// LCS dynamic-programming table. No external dependency and no git
// requirement — sufficient for human review, not meant to be machine-applied
// with `patch`/`git apply`.
func diffLines(oldLines, newLines []string) []diffLine {
	n, m := len(oldLines), len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case oldLines[i] == newLines[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	result := make([]diffLine, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldLines[i] == newLines[j]:
			result = append(result, diffLine{diffSame, oldLines[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			result = append(result, diffLine{diffRemoved, oldLines[i]})
			i++
		default:
			result = append(result, diffLine{diffAdded, newLines[j]})
			j++
		}
	}
	for ; i < n; i++ {
		result = append(result, diffLine{diffRemoved, oldLines[i]})
	}
	for ; j < m; j++ {
		result = append(result, diffLine{diffAdded, newLines[j]})
	}
	return result
}

// formatUnifiedDiff renders a diff between oldContent and newContent as a
// simple, human-readable unified-style diff labeled with path. Falls back to
// a truncation notice for very large modules (see maxDiffLines).
func formatUnifiedDiff(path, oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)

	if len(oldLines) > maxDiffLines || len(newLines) > maxDiffLines {
		fmt.Fprintf(&b, "(модуль слишком большой для построчного diff (>%d строк) — показан целиком новый вариант)\n\n", maxDiffLines)
		for _, l := range newLines {
			b.WriteString("+ " + l + "\n")
		}
		return b.String()
	}

	for _, op := range diffLines(oldLines, newLines) {
		switch op.kind {
		case diffSame:
			b.WriteString("  " + op.text + "\n")
		case diffRemoved:
			b.WriteString("- " + op.text + "\n")
		case diffAdded:
			b.WriteString("+ " + op.text + "\n")
		}
	}
	return b.String()
}
