package sync

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	gogithub "github.com/google/go-github/v84/github"
	"gopkg.in/yaml.v3"

	"github.com/elliottpolk/akctl/internal/kernel"
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
	warnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// confirmFn and warnFn are vars so tests can inject non-interactive stubs.
	confirmFn = confirm
	warnFn    = printWarning
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

	k, err := kernel.Fetch(ctx, client, owner, repo)
	if err != nil {
		return fmt.Errorf("fetch upstream kernel: %w", err)
	}
	defer os.RemoveAll(k.CacheDir)

	// Identify which local files will be touched.
	coreFiles, err := corePaths(dir, k.CacheDir)
	if err != nil {
		return fmt.Errorf("resolve core paths: %w", err)
	}

	warnFn(coreFiles)

	// Recovery mode: inform user before proceeding.
	if state == StateAgentsMDOnly || state == StateAgenticOnly {
		fmt.Println(warnStyle.Render("Partial installation detected. Sync will repair missing artifacts."))
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
// overwritten by sync.
func userOwned(rel string) bool {
	rel = filepath.ToSlash(rel)
	prefixes := []string{
		".agentic/memories/",
		".agentic/agents/kernel/memories/",
		".agentic/agents/kernel/notes/",
		".agentic/agents/kernel/references/",
		".agentic/agents/agent-foundry/memories/",
		".agentic/agents/agent-foundry/notes/",
		".agentic/agents/agent-foundry/references/",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}

// corePaths returns the list of relative paths (from dir) that sync will
// overwrite -- the intersection of upstream cache files and non-user-owned paths.
func corePaths(dir, cacheDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(cacheDir, func(src string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(cacheDir, src)
		rel = filepath.ToSlash(rel)
		if !userOwned(rel) {
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

// mergeManifest merges upstream manifest data into the local manifest.
// Rules:
//   - kernel: and core: blocks replaced entirely from upstream
//   - agents:, workflows:, skills: lists: upstream entries win for matched
//     names; local-only entries are appended
//   - project: and memories: preserved from local
func mergeManifest(local, upstream []byte) ([]byte, error) {
	var lm, um map[string]interface{}
	if err := yaml.Unmarshal(local, &lm); err != nil {
		return nil, fmt.Errorf("parse local manifest: %w", err)
	}
	if err := yaml.Unmarshal(upstream, &um); err != nil {
		return nil, fmt.Errorf("parse upstream manifest: %w", err)
	}

	if lm == nil {
		lm = make(map[string]interface{})
	}

	// Replace kernel: and core: wholly from upstream.
	for _, k := range []string{"kernel", "core"} {
		if v, ok := um[k]; ok {
			lm[k] = v
		}
	}

	// Merge list sections by name key.
	for _, section := range []string{"agents", "workflows", "skills"} {
		lm[section] = mergeLists(lm[section], um[section])
	}

	// project: and memories: stay from local -- no action needed.

	out, err := yaml.Marshal(lm)
	if err != nil {
		return nil, fmt.Errorf("marshal merged manifest: %w", err)
	}
	return out, nil
}

// mergeLists merges two YAML list values ([]interface{} of maps with a "name"
// key). Upstream values win for matched names; local-only entries are appended.
func mergeLists(local, upstream interface{}) interface{} {
	toSlice := func(v interface{}) []interface{} {
		if v == nil {
			return nil
		}
		s, _ := v.([]interface{})
		return s
	}
	getName := func(item interface{}) string {
		if m, ok := item.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				return n
			}
		}
		return ""
	}

	ls := toSlice(local)
	us := toSlice(upstream)

	// Index upstream by name.
	upIdx := make(map[string]interface{}, len(us))
	var upOrder []string
	for _, item := range us {
		n := getName(item)
		upIdx[n] = item
		upOrder = append(upOrder, n)
	}

	// Build result: upstream order first.
	result := make([]interface{}, 0, len(us)+len(ls))
	for _, n := range upOrder {
		result = append(result, upIdx[n])
	}

	// Append local-only entries.
	for _, item := range ls {
		n := getName(item)
		if _, exists := upIdx[n]; !exists {
			result = append(result, item)
		}
	}

	return result
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

// printWarning displays a destructive-overwrite warning and the list of files
// that will be changed.
func printWarning(paths []string) {
	fmt.Println(warnStyle.Render("WARNING: The following core files will be overwritten. Any local modifications will be lost:"))
	for _, p := range paths {
		fmt.Println(pathStyle.Render("  " + p))
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
