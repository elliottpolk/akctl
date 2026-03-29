package kernel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers used only in tests
func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

// --- parseFrontmatter ---

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys map[string]string
		wantErr  bool
	}{
		{
			name: "valid block",
			input: `---
title: Agentic Kernel
version: "1.0"
license: MIT
repository: https://github.com/elliottpolk/agentic-kernel
organization: The Karoshi Workshop
---

# Agentic Kernel
`,
			wantKeys: map[string]string{
				"title":        "Agentic Kernel",
				"version":      "1.0",
				"license":      "MIT",
				"repository":   "https://github.com/elliottpolk/agentic-kernel",
				"organization": "The Karoshi Workshop",
			},
		},
		{
			name: "quoted values stripped",
			input: `---
version: "1.0"
---
`,
			wantKeys: map[string]string{"version": "1.0"},
		},
		{
			name: "value with colon in it",
			input: `---
description: foo: bar
---
`,
			wantKeys: map[string]string{"description": "foo: bar"},
		},
		{
			name: "empty value",
			input: `---
name: ""
---
`,
			wantKeys: map[string]string{"name": ""},
		},
		{
			name: "missing closing delimiter",
			input: `---
title: Agentic Kernel
`,
			wantErr: true,
		},
		{
			name:    "no frontmatter at all",
			input:   "# Just a markdown file\n\nNo frontmatter here.\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFrontmatter(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			for k, v := range tt.wantKeys {
				assert.Equal(t, v, got[k], "key %q", k)
			}
		})
	}
}

// --- fetch (via mock downloader) ---

// mockDownloader implements downloader for unit tests.
type mockDownloader struct {
	tree     []string
	files    map[string][]byte
	treeErr  error
	fileErrs map[string]error
}

func (m *mockDownloader) getTree(_ context.Context) ([]string, error) {
	if m.treeErr != nil {
		return nil, m.treeErr
	}
	return m.tree, nil
}

func (m *mockDownloader) download(_ context.Context, path string) ([]byte, error) {
	if m.fileErrs != nil {
		if err, ok := m.fileErrs[path]; ok {
			return nil, err
		}
	}
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("not found: %s", path)
}

// newTestClient creates a go-github client pointed at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *github.Client {
	t.Helper()
	client := github.NewClient(nil)
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = base
	client.UploadURL = base
	return client
}

// b64 encodes s as base64 with a trailing newline (GitHub API format).
func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s)) + "\n"
}

// --- ghDownloader unit tests ---

func TestGHDownloader_GetTree(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "returns blob paths",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/git/trees/main"))
				json.NewEncoder(w).Encode(map[string]any{
					"sha": "abc123",
					"tree": []map[string]string{
						{"path": "AGENTS.md", "type": "blob"},
						{"path": ".agentic/manifest.yml", "type": "blob"},
						{"path": ".agentic/core", "type": "tree"}, // should be filtered
					},
				})
			},
			wantPaths: []string{"AGENTS.md", ".agentic/manifest.yml"},
		},
		{
			name: "API error returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			d := &ghDownloader{client: newTestClient(t, srv), owner: "testowner", repo: "testrepo"}
			paths, err := d.getTree(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPaths, paths)
		})
	}
}

func TestGHDownloader_Download(t *testing.T) {
	fileContent := "# AGENTS.MD content"

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		path        string
		wantContent string
		wantErr     bool
	}{
		{
			name: "returns file content via base64 fast path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// go-github GetContents path
				json.NewEncoder(w).Encode(map[string]any{
					"type":     "file",
					"encoding": "base64",
					"content":  b64(fileContent),
					"name":     "AGENTS.md",
				})
			},
			path:        "AGENTS.md",
			wantContent: fileContent,
		},
		{
			name: "API error returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "rate limited", http.StatusTooManyRequests)
			},
			path:    "AGENTS.md",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			d := &ghDownloader{client: newTestClient(t, srv), owner: "testowner", repo: "testrepo"}
			content, err := d.download(context.Background(), tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(content))
		})
	}
}

var minimalAgentsMD = `---
title: Agentic Kernel
version: "1.0"
license: MIT
repository: https://github.com/elliottpolk/agentic-kernel
organization: The Karoshi Workshop
---

# Agentic Kernel
`

var minimalManifest = `version: "1.0"
project:
  name: ""
`

var manifestWithAgents = `version: "1.0"
agents:
  - name: kernel
    path: .agentic/agents/kernel
  - name: personal
    path: .agentic/agents/personal
  - name: work
    path: .agentic/agents/work
`

func TestFetch(t *testing.T) {
	tests := []struct {
		name       string
		downloader *mockDownloader
		wantErr    bool
		check      func(t *testing.T, k *KernelInfo)
	}{
		{
			name: "success -- AGENTS.md and .agentic files fetched",
			downloader: &mockDownloader{
				tree: []string{
					"AGENTS.md",
					".agentic/manifest.yml",
					"README.md", // should be filtered out
				},
				files: map[string][]byte{
					"AGENTS.md":             []byte(minimalAgentsMD),
					".agentic/manifest.yml": []byte(minimalManifest),
				},
			},
			check: func(t *testing.T, k *KernelInfo) {
				assert.Equal(t, "1.0", k.Version)
				assert.Equal(t, "Agentic Kernel", k.Title)
				assert.Equal(t, "MIT", k.License)
				assert.NotEmpty(t, k.CacheDir)

				// verify AGENTS.md was written to cache
				content, err := readFile(k.CacheDir + "/AGENTS.md")
				require.NoError(t, err)
				assert.Contains(t, string(content), "Agentic Kernel")
			},
		},
		{
			name: "tree fetch failure returns error",
			downloader: &mockDownloader{
				treeErr: fmt.Errorf("connection refused"),
			},
			wantErr: true,
		},
		{
			name: "file download failure returns error and cleans cache",
			downloader: &mockDownloader{
				tree: []string{"AGENTS.md"},
				fileErrs: map[string]error{
					"AGENTS.md": fmt.Errorf("rate limited"),
				},
			},
			wantErr: true,
		},
		{
			name: "AGENTS.md missing from tree returns error",
			downloader: &mockDownloader{
				tree:  []string{".agentic/manifest.yml"},
				files: map[string][]byte{".agentic/manifest.yml": []byte(minimalManifest)},
			},
			wantErr: true,
		},
		{
			name: "AGENTS.md with bad frontmatter returns error",
			downloader: &mockDownloader{
				tree:  []string{"AGENTS.md"},
				files: map[string][]byte{"AGENTS.md": []byte("# No frontmatter here")},
			},
			wantErr: true,
		},
		{
			name: ".gitkeep files are skipped without download but directory is created",
			downloader: &mockDownloader{
				tree: []string{
					"AGENTS.md",
					".agentic/manifest.yml",
					".agentic/agents/kernel/memories/.gitkeep",
					".agentic/agents/kernel/notes/.gitkeep",
				},
				files: map[string][]byte{
					"AGENTS.md":             []byte(minimalAgentsMD),
					".agentic/manifest.yml": []byte(minimalManifest),
					// no entries for .gitkeep -- download must not be called for them
				},
			},
			check: func(t *testing.T, k *KernelInfo) {
				assert.NotEmpty(t, k.CacheDir)
				// directories should exist even though .gitkeep was not downloaded
				_, err := os.Stat(filepath.Join(k.CacheDir, ".agentic", "agents", "kernel", "memories"))
				assert.NoError(t, err, "memories dir should be created")
				_, err = os.Stat(filepath.Join(k.CacheDir, ".agentic", "agents", "kernel", "notes"))
				assert.NoError(t, err, "notes dir should be created")
				// the .gitkeep files themselves should not exist in cache
				_, err = os.Stat(filepath.Join(k.CacheDir, ".agentic", "agents", "kernel", "memories", ".gitkeep"))
				assert.True(t, os.IsNotExist(err), ".gitkeep should not be written to cache")
			},
		},
		{
			name: "non-kernel paths filtered out",
			downloader: &mockDownloader{
				tree: []string{
					"AGENTS.md",
					"README.md",
					"LICENSE.md",
					".github/workflows/ci.yml",
					".agentic/manifest.yml",
				},
				files: map[string][]byte{
					"AGENTS.md":             []byte(minimalAgentsMD),
					".agentic/manifest.yml": []byte(minimalManifest),
				},
			},
			check: func(t *testing.T, k *KernelInfo) {
				assert.NotEmpty(t, k.CacheDir)
				// README.md and LICENSE.md should NOT be in the cache
				_, err := readFile(k.CacheDir + "/README.md")
				assert.Error(t, err, "README.md should not be cached")
			},
		},
		{
			name: "agent paths populated from manifest",
			downloader: &mockDownloader{
				tree: []string{
					"AGENTS.md",
					".agentic/manifest.yml",
				},
				files: map[string][]byte{
					"AGENTS.md":             []byte(minimalAgentsMD),
					".agentic/manifest.yml": []byte(manifestWithAgents),
				},
			},
			check: func(t *testing.T, k *KernelInfo) {
				assert.Equal(t, []string{
					".agentic/agents/kernel",
					".agentic/agents/personal",
					".agentic/agents/work",
				}, k.AgentPaths)
			},
		},
		{
			name: "agent paths empty when manifest has no agents",
			downloader: &mockDownloader{
				tree: []string{
					"AGENTS.md",
					".agentic/manifest.yml",
				},
				files: map[string][]byte{
					"AGENTS.md":             []byte(minimalAgentsMD),
					".agentic/manifest.yml": []byte(minimalManifest),
				},
			},
			check: func(t *testing.T, k *KernelInfo) {
				assert.Empty(t, k.AgentPaths)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := fetch(context.Background(), tt.downloader)

			if tt.wantErr {
				assert.Error(t, err)
				// cache dir should be cleaned up on error
				if k != nil {
					_, statErr := readFile(k.CacheDir)
					assert.True(t, statErr != nil || k.CacheDir == "", "cache dir should not remain on error")
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, k)
			defer os.RemoveAll(k.CacheDir)

			if tt.check != nil {
				tt.check(t, k)
			}
		})
	}
}

// --- AgentPathsFromManifest ---

func TestAgentPathsFromManifest(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "returns agent paths from manifest",
			content: manifestWithAgents,
			want: []string{
				".agentic/agents/kernel",
				".agentic/agents/personal",
				".agentic/agents/work",
			},
		},
		{
			name:    "no agents returns empty slice",
			content: minimalManifest,
			want:    []string{},
		},
		{
			name:    "agents with empty path skipped",
			content: "agents:\n  - name: broken\n    path: \"\"\n  - name: ok\n    path: .agentic/agents/ok\n",
			want:    []string{".agentic/agents/ok"},
		},
		{
			name:    "file not found returns error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "manifest.yml")
			if tt.content != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.content), 0644))
			}
			got, err := AgentPathsFromManifest(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// --- ManifestRepo ---

func TestManifestRepo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			content: `kernel:
  repository: https://github.com/elliottpolk/agentic-kernel
  version: "1.0"
`,
			want: "https://github.com/elliottpolk/agentic-kernel",
		},
		{
			name:    "missing kernel.repository",
			content: "version: \"1.0\"\n",
			wantErr: true,
		},
		{
			name:    "empty repository field",
			content: "kernel:\n  repository: \"\"\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agentic"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte(tt.content), 0644))
			got, err := ManifestRepo(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- AgentsRepo ---

func TestAgentsRepo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "valid",
			content: minimalAgentsMD,
			want:    "https://github.com/elliottpolk/agentic-kernel",
		},
		{
			name:    "no frontmatter",
			content: "# Agentic Kernel\n",
			wantErr: true,
		},
		{
			name: "missing repository field",
			content: `---
title: Agentic Kernel
version: "1.0"
---
`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(tt.content), 0644))
			got, err := AgentsRepo(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
