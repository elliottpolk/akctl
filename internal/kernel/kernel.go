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
	"gopkg.in/yaml.v3"
)

// Manifest is a minimal typed view of .agentic/manifest.yml sufficient for
// sync operations. Unknown/user fields are preserved in the raw decoded map.
type Manifest struct {
	Kernel struct {
		Repository string `yaml:"repository"`
		Version    string `yaml:"version"`
	} `yaml:"kernel"`
}

// manifestRepo reads .agentic/manifest.yml from targetDir and returns the
// kernel.repository value, or an error if absent or unreadable.
func ManifestRepo(targetDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, ".agentic", "manifest.yml"))
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("parse manifest: %w", err)
	}
	if strings.TrimSpace(m.Kernel.Repository) == "" {
		return "", fmt.Errorf("manifest missing kernel.repository")
	}
	return m.Kernel.Repository, nil
}

// AgentsRepo reads AGENTS.md from targetDir and returns the repository field
// from its frontmatter, or an error if the file is absent or malformed.
func AgentsRepo(targetDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, "AGENTS.md"))
	if err != nil {
		return "", fmt.Errorf("read AGENTS.md: %w", err)
	}
	fm, err := parseFrontmatter(string(data))
	if err != nil {
		return "", fmt.Errorf("parse AGENTS.md frontmatter: %w", err)
	}
	repo := strings.TrimSpace(fm["repository"])
	if repo == "" {
		return "", fmt.Errorf("AGENTS.md missing repository field")
	}
	return repo, nil
}

const ref = "main"

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

// downloader abstracts GitHub file retrieval so it can be swapped in tests.
type downloader interface {
	getTree(ctx context.Context) ([]string, error)
	download(ctx context.Context, path string) ([]byte, error)
}

// ghDownloader is the real GitHub-backed downloader.
type ghDownloader struct {
	client *github.Client
	owner  string
	repo   string
}

func (d *ghDownloader) getTree(ctx context.Context) ([]string, error) {
	tree, _, err := d.client.Git.GetTree(ctx, d.owner, d.repo, ref, true)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range tree.Entries {
		if entry.Type == nil || *entry.Type != "blob" || entry.Path == nil {
			continue
		}
		paths = append(paths, *entry.Path)
	}
	return paths, nil
}

func (d *ghDownloader) download(ctx context.Context, path string) ([]byte, error) {
	rc, resp, err := d.client.Repositories.DownloadContents(ctx, d.owner, d.repo, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(rc)
}

// Fetch retrieves the upstream kernel from GitHub, writes all relevant files
// to a temporary directory, and returns a KernelInfo with metadata and the
// cache path. On any failure the temp dir is removed and the error is returned
// verbatim. The caller is responsible for os.RemoveAll(k.CacheDir) when done.
func Fetch(ctx context.Context, client *github.Client, owner, repo string) (*KernelInfo, error) {
	return fetch(ctx, &ghDownloader{client: client, owner: owner, repo: repo})
}

// fetch is the testable core of Fetch -- accepts any downloader implementation.
func fetch(ctx context.Context, d downloader) (*KernelInfo, error) {
	paths, err := d.getTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch kernel tree: %w", err)
	}

	cacheDir, err := os.MkdirTemp("", "akctl-kernel-*")
	if err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	var agentsMDContent []byte

	for _, p := range paths {
		if p != "AGENTS.md" && !strings.HasPrefix(p, ".agentic/") {
			continue
		}

		content, err := d.download(ctx, p)
		if err != nil {
			os.RemoveAll(cacheDir)
			return nil, fmt.Errorf("fetch %s: %w", p, err)
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
