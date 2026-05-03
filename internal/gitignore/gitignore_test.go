package gitignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- containsLine ---

func TestContainsLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    string
		want    bool
	}{
		{"empty content", "", Pattern, false},
		{"exact match", Pattern + "\n", Pattern, true},
		{"match with surrounding whitespace in file", "  " + Pattern + "  \n", Pattern, true},
		{"substring not matched", "foo/" + Pattern + "\n", Pattern, false},
		{"multiline - present", "node_modules/\n.DS_Store\n" + Pattern + "\n", Pattern, true},
		{"multiline - absent", "node_modules/\n.DS_Store\n", Pattern, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsLine(tt.content, tt.line))
		})
	}
}

// --- Ensure ---

func TestEnsure(t *testing.T) {
	tests := []struct {
		name         string
		existing     string // "" means no .gitignore file
		patterns     []string
		wantContains []string
		wantExact    string // when set, the full file content must match exactly
	}{
		{
			name:     "creates file with pattern when none exists",
			existing: "",
			patterns: []string{Pattern},
			wantContains: []string{
				sectionHeader,
				Pattern,
			},
		},
		{
			name:     "appends pattern and header to existing file",
			existing: "node_modules/\n.DS_Store\n",
			patterns: []string{Pattern},
			wantContains: []string{
				"node_modules/",
				".DS_Store",
				sectionHeader,
				Pattern,
			},
		},
		{
			name:      "idempotent when pattern already present",
			existing:  sectionHeader + "\n" + Pattern + "\n",
			patterns:  []string{Pattern},
			wantExact: sectionHeader + "\n" + Pattern + "\n",
		},
		{
			name:     "adds newline before section when file lacks trailing newline",
			existing: "node_modules/",
			patterns: []string{Pattern},
			wantContains: []string{
				"node_modules/",
				Pattern,
			},
		},
		{
			name:     "multiple patterns added together",
			existing: "",
			patterns: []string{Pattern, "*.secret"},
			wantContains: []string{
				sectionHeader,
				Pattern,
				"*.secret",
			},
		},
		{
			name:     "only missing patterns added when some already present",
			existing: Pattern + "\n",
			patterns: []string{Pattern, "*.secret"},
			wantContains: []string{
				Pattern,
				"*.secret",
			},
		},
		{
			name:     "header not duplicated when already present",
			existing: sectionHeader + "\n",
			patterns: []string{Pattern},
			wantContains: []string{
				Pattern,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.existing != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.existing), 0644))
			}

			err := Ensure(dir, tt.patterns...)
			require.NoError(t, err)

			got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
			require.NoError(t, err)

			if tt.wantExact != "" {
				assert.Equal(t, tt.wantExact, string(got))
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, string(got), want)
			}
		})
	}
}

func TestEnsure_headerNotDuplicated(t *testing.T) {
	dir := t.TempDir()

	// Call Ensure twice; the header should appear only once.
	require.NoError(t, Ensure(dir, Pattern))
	require.NoError(t, Ensure(dir, Pattern))

	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)

	count := 0
	for _, line := range splitLines(string(got)) {
		if line == sectionHeader {
			count++
		}
	}
	assert.Equal(t, 1, count, "section header should appear exactly once")
}

func splitLines(s string) []string {
	var out []string
	for _, l := range []byte(s) {
		_ = l
	}
	// Simple split without importing strings in test helpers.
	start := 0
	for i, c := range s {
		if c == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
