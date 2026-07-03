package tools

import (
	"strings"
	"testing"
)

func TestFormatUnifiedDiff(t *testing.T) {
	old := "Строка1\nСтрока2\nСтрока3\n"
	new := "Строка1\nИзменено\nСтрока3\n"

	result := formatUnifiedDiff("path/to/module.bsl", old, new)

	for _, want := range []string{
		"--- a/path/to/module.bsl",
		"+++ b/path/to/module.bsl",
		"- Строка2",
		"+ Изменено",
		"  Строка1",
		"  Строка3",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("expected diff to contain %q, got:\n%s", want, result)
		}
	}
}

func TestFormatUnifiedDiff_NoChange(t *testing.T) {
	content := "Строка1\nСтрока2\n"
	result := formatUnifiedDiff("m.bsl", content, content)

	if strings.Contains(result, "-") || strings.Contains(result, "+ ") {
		// Only the --- / +++ header lines should contain a leading dash/plus.
		lines := strings.Split(result, "\n")
		for _, l := range lines[2:] {
			if strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "+ ") {
				t.Errorf("expected no added/removed lines for identical content, got line: %q", l)
			}
		}
	}
}

func TestFormatUnifiedDiff_LargeModuleFallback(t *testing.T) {
	var b strings.Builder
	for i := 0; i < maxDiffLines+10; i++ {
		b.WriteString("line\n")
	}
	old := b.String()
	newContent := "полностью новый маленький модуль\n"

	result := formatUnifiedDiff("big.bsl", old, newContent)
	if !strings.Contains(result, "слишком большой") {
		t.Errorf("expected large-module fallback notice, got:\n%s", result[:200])
	}
	if !strings.Contains(result, "полностью новый маленький модуль") {
		t.Error("expected fallback to still show the new content")
	}
}
