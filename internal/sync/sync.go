package sync

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	gogithub "github.com/google/go-github/v84/github"
	"gopkg.in/yaml.v3"

	"github.com/elliottpolk/akctl/internal/kernel"
	"github.com/elliottpolk/akctl/internal/ui"
)

// State represents the detected installation state of the agentic kernel.
type State int

const (
	StateAbsent       State = iota // neither AGENTS.md nor .agentic/ present or valid
	StateAgentsMDOnly              // valid AGENTS.md only; .agentic/ missing
	StateAgenticOnly               // valid .agentic/ only; AGENTS.md missing
	StateBoth                      // both present and valid (normal case)
)

// Options controls sync behavior.
type Options struct {
	Force     bool
	TargetDir string
}

var (
	errAbsentToInit = fmt.Errorf("no valid kernel installation found; run 'akctl init' to initialize")
	errAborted      = fmt.Errorf("aborted")
)

var (
	// confirmFn and warnFn are vars so tests can inject non-interactive stubs.
	confirmFn = confirm
	warnFn    = func(source string, paths []string) { printWarning(source, paths) }
)

// Run is the top-level sync orchestration entry point.
func Run(ctx context.Context, client *gogithub.Client, opts Options) error {
	dir := opts.TargetDir
	if dir == "" {
		dir = "."
	}

	state, repoURL, err := detect(dir)
	if err != nil {
		return fmt.Errorf("detect state: %w", err)
	}
	if state == StateAbsent {
		return errAbsentToInit
	}

	owner, repo, err := splitRepo(repoURL)
	if err != nil {
		return fmt.Errorf("parse repository: %w", err)
	}

	var reporter kernel.ProgressReporter
	if !opts.Force {
		reporter = ui.NewTeaProgressReporter()
	}

	k, err := kernel.Fetch(ctx, client, owner, repo, reporter)
	if err != nil {
		return fmt.Errorf("fetch upstream kernel: %w", err)
	}
	defer os.RemoveAll(k.CacheDir)

	// Identify which local files will be touched, using the upstream manifest's
	// agent list to dynamically determine user-owned paths.
	coreFiles, err := corePaths(dir, k.CacheDir, k.AgentPaths)
	if err != nil {
		return fmt.Errorf("resolve core paths: %w", err)
	}

	warnFn(repoURL, coreFiles)

	// Recovery mode: inform user before proceeding.
	if state == StateAgentsMDOnly || state == StateAgenticOnly {
		fmt.Println(ui.WarnStyle.Render("Partial installation detected. Sync will repair missing artifacts."))
	}

	ok, err := confirmFn(opts.Force)
	if err != nil {
		return fmt.Errorf("confirm: %w", err)
	}
	if !ok {
		return errAborted
	}

	// Cache current local state before any writes.
	cache, err := cacheFiles(dir, coreFiles)
	if err != nil {
		return fmt.Errorf("cache local files: %w", err)
	}

	if err := writeCore(dir, k.CacheDir, coreFiles); err != nil {
		if rbErr := rollback(cache, dir); rbErr != nil {
			return fmt.Errorf("write failed (%v); rollback also failed: %w", err, rbErr)
		}
		return fmt.Errorf("sync failed, rolled back: %w", err)
	}

	os.RemoveAll(cache)
	return nil
}

// detect inspects the target directory and returns the install State and
// the resolved upstream repository URL.
func detect(dir string) (State, string, error) {
	agentsOk, agentsRepo := probeAgents(dir)
	agenticOk, agenticRepo := probeAgentic(dir)

	switch {
	case agentsOk && agenticOk:
		// Normal case: prefer manifest repo.
		return StateBoth, agenticRepo, nil
	case agentsOk && !agenticOk:
		return StateAgentsMDOnly, agentsRepo, nil
	case !agentsOk && agenticOk:
		return StateAgenticOnly, agenticRepo, nil
	default:
		return StateAbsent, "", nil
	}
}

// probeAgents returns (valid, repository) for AGENTS.md in dir.
func probeAgents(dir string) (bool, string) {
	repo, err := kernel.AgentsRepo(dir)
	return err == nil && repo != "", repo
}

// probeAgentic returns (valid, repository) for .agentic/manifest.yml in dir.
func probeAgentic(dir string) (bool, string) {
	repo, err := kernel.ManifestRepo(dir)
	return err == nil && repo != "", repo
}

// splitRepo splits a full repository URL (e.g. https://github.com/owner/repo
// or github.com/owner/repo) into owner and repo.
func splitRepo(repoURL string) (string, string, error) {
	s := repoURL
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	parts := strings.Split(strings.Trim(s, "/"), "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("cannot parse repository URL: %q", repoURL)
	}
	return parts[1], parts[2], nil
}

// userOwned returns true for paths that belong to the user and must not be
// overwritten by sync. agentPaths is the list of agent directory paths
// declared in the upstream manifest (e.g. ".agentic/agents/kernel").
func userOwned(rel string, agentPaths []string) bool {
	rel = filepath.ToSlash(rel)
	// Global memories dir is always user-owned regardless of kernel structure.
	if strings.HasPrefix(rel, ".agentic/memories/") {
		return true
	}
	// Per-agent subdirectories that contain user data.
	for _, p := range agentPaths {
		p = strings.TrimSuffix(filepath.ToSlash(p), "/")
		if strings.HasPrefix(rel, p+"/memories/") ||
			strings.HasPrefix(rel, p+"/notes/") ||
			strings.HasPrefix(rel, p+"/references/") {
			return true
		}
	}
	return false
}

// corePaths returns the list of relative paths (from dir) that sync will
// overwrite -- the intersection of upstream cache files and non-user-owned paths.
// agentPaths is forwarded to userOwned to derive protected paths dynamically.
func corePaths(dir, cacheDir string, agentPaths []string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(cacheDir, func(src string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(cacheDir, src)
		rel = filepath.ToSlash(rel)
		if !userOwned(rel, agentPaths) {
			paths = append(paths, rel)
		}
		return nil
	})
	return paths, err
}

// cacheFiles copies all files in paths (relative to dir) to a new temp dir.
// Returns the temp dir path. Only copies files that already exist locally.
func cacheFiles(dir string, paths []string) (string, error) {
	tmp, err := os.MkdirTemp("", "akctl-sync-*")
	if err != nil {
		return "", err
	}
	for _, rel := range paths {
		src := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := os.ReadFile(src)
		if os.IsNotExist(err) {
			continue // file doesn't exist locally yet; nothing to cache
		}
		if err != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("cache %s: %w", rel, err)
		}
		dst := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			os.RemoveAll(tmp)
			return "", err
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("cache write %s: %w", rel, err)
		}
	}
	return tmp, nil
}

// writeCore copies upstream core files from cacheDir to dir, routing
// manifest.yml through mergeManifest.
func writeCore(dir, cacheDir string, paths []string) error {
	for _, rel := range paths {
		src := filepath.Join(cacheDir, filepath.FromSlash(rel))
		dst := filepath.Join(dir, filepath.FromSlash(rel))

		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read upstream %s: %w", rel, err)
		}

		if rel == filepath.ToSlash(filepath.Join(".agentic", "manifest.yml")) {
			localData, err := os.ReadFile(dst)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("read local manifest: %w", err)
			}
			if localData != nil {
				data, err = mergeManifest(localData, data)
				if err != nil {
					return fmt.Errorf("merge manifest: %w", err)
				}
			}
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}

// mergeManifest merges upstream manifest data into the local manifest using
// gopkg.in/yaml.v3 Node trees so that comments and key ordering of the local
// file are always preserved in the output.
//
// Rules:
//   - kernel: and core: nodes replaced entirely from upstream
//   - agents:, workflows:, skills: lists merged by name; upstream wins for
//     matched names; local-only entries appended
//   - project: and memories: nodes kept from local untouched
func mergeManifest(local, upstream []byte) ([]byte, error) {
	var localDoc yaml.Node
	if err := yaml.Unmarshal(local, &localDoc); err != nil {
		return nil, fmt.Errorf("parse local manifest: %w", err)
	}
	var upstreamDoc yaml.Node
	if err := yaml.Unmarshal(upstream, &upstreamDoc); err != nil {
		return nil, fmt.Errorf("parse upstream manifest: %w", err)
	}
	if len(localDoc.Content) == 0 || len(upstreamDoc.Content) == 0 {
		return nil, fmt.Errorf("empty manifest document")
	}

	lmap := localDoc.Content[0]
	umap := upstreamDoc.Content[0]

	if lmap.Kind != yaml.MappingNode || umap.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("manifest must be a YAML mapping at its root")
	}

	for _, key := range []string{"kernel", "core"} {
		replaceNode(lmap, umap, key)
	}
	for _, key := range []string{"agents", "workflows", "skills"} {
		mergeSeq(lmap, umap, key)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&localDoc); err != nil {
		return nil, fmt.Errorf("marshal merged manifest: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close yaml encoder: %w", err)
	}
	return buf.Bytes(), nil
}

// nodeValue finds the value node for a key in a yaml.v3 mapping node, or nil.
func nodeValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// replaceNode replaces key's value in lmap with the value from umap.
// If the key is absent in lmap, the upstream key-value pair is appended.
// No-op if the key is absent from umap.
func replaceNode(lmap, umap *yaml.Node, key string) {
	uval := nodeValue(umap, key)
	if uval == nil {
		return
	}
	for i := 0; i+1 < len(lmap.Content); i += 2 {
		if lmap.Content[i].Value == key {
			lmap.Content[i+1] = uval
			return
		}
	}
	// Key not in local; find and append the full upstream key-value pair.
	for i := 0; i+1 < len(umap.Content); i += 2 {
		if umap.Content[i].Value == key {
			lmap.Content = append(lmap.Content, umap.Content[i], umap.Content[i+1])
			return
		}
	}
}

// mergeSeq merges a sequence in lmap with one from umap by "name" key.
// Upstream items win for matched names; local-only items are appended.
func mergeSeq(lmap, umap *yaml.Node, key string) {
	uval := nodeValue(umap, key)
	if uval == nil || uval.Kind != yaml.SequenceNode {
		return
	}

	lval := nodeValue(lmap, key)
	if lval == nil {
		for i := 0; i+1 < len(umap.Content); i += 2 {
			if umap.Content[i].Value == key {
				lmap.Content = append(lmap.Content, umap.Content[i], umap.Content[i+1])
				return
			}
		}
		return
	}
	if lval.Kind != yaml.SequenceNode {
		lval.Content = uval.Content
		return
	}

	upNames := make(map[string]bool, len(uval.Content))
	for _, item := range uval.Content {
		if n := seqItemName(item); n != "" {
			upNames[n] = true
		}
	}

	merged := make([]*yaml.Node, 0, len(uval.Content)+len(lval.Content))
	merged = append(merged, uval.Content...)
	for _, item := range lval.Content {
		if n := seqItemName(item); n != "" && !upNames[n] {
			merged = append(merged, item)
		}
	}
	lval.Content = merged
}

// seqItemName returns the "name" field value from a sequence item mapping node.
func seqItemName(item *yaml.Node) string {
	if item.Kind != yaml.MappingNode {
		return ""
	}
	val := nodeValue(item, "name")
	if val == nil {
		return ""
	}
	return val.Value
}

// rollback restores all files from cacheDir back to dir.
func rollback(cacheDir, dir string) error {
	err := filepath.WalkDir(cacheDir, func(src string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(cacheDir, src)
		dst := filepath.Join(dir, rel)
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(cacheDir)
}

// printWarning displays the upstream kernel source and the list of files
// that will be destructively overwritten.
func printWarning(source string, paths []string) {
	fmt.Println(ui.WarnStyle.Render("WARNING: Syncing from " + source))
	fmt.Println(ui.WarnStyle.Render("The following core files will be overwritten. Any local modifications will be lost:"))
	for _, p := range paths {
		fmt.Println(ui.PathStyle.Render("  " + p))
	}
	fmt.Println()
}

// confirm prompts for explicit confirmation unless force is set.
func confirm(force bool) (bool, error) {
	if force {
		return true, nil
	}
	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Core files will be permanently overwritten. Proceed?").
				Affirmative("Yes").
				Negative("No").
				Value(&ok),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return ok, nil
}
