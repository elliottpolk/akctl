// scaffold: generated from docs/specs/init-subcommand.md
package kernel

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v84/github"
)

const (
	owner = "elliottpolk"
	repo  = "agentic-kernel"
	ref   = "main"
)

// KernelInfo holds metadata parsed from the upstream AGENTS.md frontmatter
// and the path to the local temp cache of fetched files.
type KernelInfo struct {
	Version      string
	Title        string
	Repository   string
	Organization string
	License      string
	CacheDir     string
}

// Fetch retrieves the upstream kernel from GitHub, writes all relevant files
// to a temporary directory, and returns a KernelInfo with metadata and the
// cache path. On any failure the temp dir is removed and the error is returned
// verbatim. The caller is responsible for os.RemoveAll(k.CacheDir) when done.
func Fetch(ctx context.Context) (*KernelInfo, error) {
	client := github.NewClient(nil)

	tree, _, err := client.Git.GetTree(ctx, owner, repo, ref, true)
	if err != nil {
		return nil, fmt.Errorf("fetch kernel tree: %w", err)
	}

	cacheDir, err := os.MkdirTemp("", "akctl-kernel-*")
	if err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	var agentsMDContent []byte

	for _, entry := range tree.Entries {
		if entry.Type == nil || *entry.Type != "blob" {
			continue
		}
		if entry.Path == nil {
			continue
		}
		p := *entry.Path
		if p != "AGENTS.md" && !strings.HasPrefix(p, ".agentic/") {
			continue
		}

		rc, resp, err := client.Repositories.DownloadContents(ctx, owner, repo, p, &github.RepositoryContentGetOptions{Ref: ref})
		if err != nil {
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("fetch %s: %w", p, err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			rc.Close()
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("fetch %s: unexpected status %d", p, resp.StatusCode)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("read %s: %w", p, err)
		}

		dest := filepath.Join(cacheDir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("create cache subdir for %s: %w", p, err)
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("cache %s: %w", p, err)
		}

		if p == "AGENTS.md" {
			agentsMDContent = content
		}
	}

	if agentsMDContent == nil {
		os.RemoveAll(cacheDir)
		return nil, fmt.Errorf("AGENTS.md not found in upstream kernel")
	}

	fm, err := parseFrontmatter(string(agentsMDContent))
	if err != nil {
		os.RemoveAll(cacheDir)
		return nil, fmt.Errorf("parse AGENTS.md frontmatter: %w", err)
	}

	return &KernelInfo{
		Version:      fm["version"],
		Title:        fm["title"],
		Repository:   fm["repository"],
		Organization: fm["organization"],
		License:      fm["license"],
		CacheDir:     cacheDir,
	}, nil
}

// parseFrontmatter extracts key/value pairs from YAML-style frontmatter
// delimited by "---" lines. Values are unquoted if wrapped in double quotes.
func parseFrontmatter(content string) (map[string]string, error) {
	result := make(map[string]string)

	lines := strings.Split(content, "\n")
	inFront := false
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			count++
			if count == 1 {
				inFront = true
				continue
			}
			break
		}
		if !inFront {
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])
		val = strings.Trim(val, `"`)
		result[key] = val
	}

	if count < 2 {
		return nil, fmt.Errorf("no valid frontmatter block found")
	}

	return result, nil
}
