package gitignore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Pattern is the gitignore glob that shields per-clone local override files
// from being committed to a shared repository.
const Pattern = ".agentic/**/*.local.md"

const sectionHeader = "# agentic kernel - local overrides"

// Ensure adds any of the given patterns not already present in
// targetDir/.gitignore. If the file does not exist it is created.
// Patterns are grouped under a section comment on first addition.
func Ensure(targetDir string, patterns ...string) error {
	path := filepath.Join(targetDir, ".gitignore")

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}
	content := string(data)

	var toAdd []string
	for _, p := range patterns {
		if !containsLine(content, p) {
			toAdd = append(toAdd, p)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	var sb strings.Builder
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		sb.WriteByte('\n')
	}
	if !containsLine(content, sectionHeader) {
		sb.WriteString("\n" + sectionHeader + "\n")
	}
	for _, p := range toAdd {
		sb.WriteString(p + "\n")
	}

	if err := os.WriteFile(path, []byte(content+sb.String()), 0644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}

// containsLine reports whether content contains line as a standalone line
// (exact match after trimming surrounding whitespace).
func containsLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}
