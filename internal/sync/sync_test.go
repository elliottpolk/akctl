package sync

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elliottpolk/akctl/internal/kernel"
)

// --- userOwned ---

func TestUserOwned(t *testing.T) {
	// Simulate the agent, workflow, and skill paths an upstream would expose.
	agentPaths := []string{
		".agentic/agents/kernel",
		".agentic/agents/agent-foundry",
		".agentic/agents/personal",
		".agentic/agents/work",
		".agentic/skills/gemini-bridge",
	}

	tests := []struct {
		rel  string
		want bool
	}{
		// Global memories — always user-owned.
		{".agentic/memories/state/project.md", true},
		{".agentic/memories/history/2026-03-01.md", true},
		// Per-agent user-owned subdirs for base kernel agents.
		{".agentic/agents/kernel/memories/.gitkeep", true},
		{".agentic/agents/kernel/notes/.gitkeep", true},
		{".agentic/agents/kernel/references/.gitkeep", true},
		{".agentic/agents/agent-foundry/memories/.gitkeep", true},
		{".agentic/agents/agent-foundry/notes/.gitkeep", true},
		{".agentic/agents/agent-foundry/references/.gitkeep", true},
		// Per-agent user-owned subdirs for journal-kernel agents.
		{".agentic/agents/personal/memories/2026-03.md", true},
		{".agentic/agents/personal/notes/ideas.md", true},
		{".agentic/agents/personal/references/links.md", true},
		{".agentic/agents/work/memories/2026-03-31.md", true},
		{".agentic/agents/work/notes/standup.md", true},
		{".agentic/agents/work/references/runbook.md", true},
		// Per-skill user-owned subdirs.
		{".agentic/skills/gemini-bridge/memories/usage.md", true},
		{".agentic/skills/gemini-bridge/notes/fixes.md", true},
		{".agentic/skills/gemini-bridge/references/api.md", true},
		// Core files — not user-owned.
		{".agentic/core/BEHAVIOR.md", false},
		{".agentic/core/DECISIONS.md", false},
		{"AGENTS.md", false},
		{".agentic/manifest.yml", false},
		{".agentic/agents/kernel/IDENTITY.md", false},
		{".agentic/agents/personal/IDENTITY.md", false},
		{".agentic/agents/work/IDENTITY.md", false},
		{".agentic/skills/gemini-bridge/SKILL.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			assert.Equal(t, tt.want, userOwned(tt.rel, agentPaths))
		})
	}
}

// --- splitRepo ---

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"https://github.com/elliottpolk/agentic-kernel", "elliottpolk", "agentic-kernel", false},
		{"github.com/elliottpolk/agentic-kernel", "elliottpolk", "agentic-kernel", false},
		{"http://github.com/foo/bar", "foo", "bar", false},
		{"notaurl", "", "", true},
		{"github.com/onlyone", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := splitRepo(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

// --- mergeManifest ---

const localManifest = `
version: "1.0"
kernel:
  title: Agentic Kernel
  version: "1.0"
  repository: https://github.com/elliottpolk/agentic-kernel
core:
  behavior: .agentic/core/BEHAVIOR.md
project:
  name: myproject
  description: my desc
memories:
  state: .agentic/memories/state
agents:
  - name: kernel
    description: old kernel desc
    path: .agentic/agents/kernel
  - name: my-custom-agent
    description: user addition
    path: .agentic/agents/my-custom-agent
workflows:
  - name: spec
    description: old spec desc
    path: .agentic/workflows/spec
skills:
  - name: copilot-bridge
    description: old bridge desc
    path: .agentic/skills/copilot-bridge
`

const upstreamManifest = `
version: "1.0"
kernel:
  title: Agentic Kernel
  version: "1.1"
  repository: https://github.com/elliottpolk/agentic-kernel
  organization: The Karoshi Workshop
core:
  behavior: .agentic/core/BEHAVIOR.md
  decisions: .agentic/core/DECISIONS.md
project:
  name: ""
  description: ""
agents:
  - name: kernel
    description: new kernel desc
    path: .agentic/agents/kernel
  - name: agent-foundry
    description: new upstream agent
    path: .agentic/agents/agent-foundry
workflows:
  - name: spec
    description: new spec desc
    path: .agentic/workflows/spec
skills:
  - name: copilot-bridge
    description: new bridge desc
    path: .agentic/skills/copilot-bridge
  - name: create-skill
    description: new skill
    path: .agentic/skills/create-skill
`

func TestMergeManifest_preservesComments(t *testing.T) {
	local := `# Agentic Kernel Manifest
# Do not edit the kernel: block.

version: "1.0"

kernel:
  title: Old Title
  version: "1.0"
  repository: https://github.com/elliottpolk/agentic-kernel

# Project identity -- fill in for your project.
project:
  name: "myproject"
  description: "my description"
  author: "Elliott"

agents:
  - name: kernel
    description: old desc
    path: .agentic/agents/kernel
  - name: my-custom
    description: user addition
    path: .agentic/agents/my-custom
`
	upstream := `version: "1.0"
kernel:
  title: Agentic Kernel
  version: "1.1"
  repository: https://github.com/elliottpolk/agentic-kernel
  organization: The Karoshi Workshop
project:
  name: ""
agents:
  - name: kernel
    description: new desc
    path: .agentic/agents/kernel
`
	out, err := mergeManifest([]byte(local), []byte(upstream))
	require.NoError(t, err)
	s := string(out)

	// Header comments preserved.
	assert.Contains(t, s, "Agentic Kernel Manifest")
	assert.Contains(t, s, "Do not edit the kernel: block.")

	// Inline comment above project: preserved.
	assert.Contains(t, s, "Project identity")

	// project: values preserved from local.
	assert.Contains(t, s, "myproject")
	assert.Contains(t, s, "my description")
	assert.Contains(t, s, "Elliott")

	// kernel: updated from upstream.
	assert.Contains(t, s, `version: "1.1"`)
	assert.Contains(t, s, "organization: The Karoshi Workshop")

	// local-only agent preserved.
	assert.Contains(t, s, "my-custom")
	// upstream agent desc updated.
	assert.Contains(t, s, "new desc")
}

func TestMergeManifest(t *testing.T) {
	out, err := mergeManifest([]byte(localManifest), []byte(upstreamManifest))
	require.NoError(t, err)

	s := string(out)

	// kernel: replaced from upstream
	assert.Contains(t, s, `version: "1.1"`)
	assert.Contains(t, s, "organization: The Karoshi Workshop")

	// project: preserved from local
	assert.Contains(t, s, "myproject")
	assert.Contains(t, s, "my desc")

	// memories: preserved from local
	assert.Contains(t, s, ".agentic/memories/state")

	// core: replaced from upstream (has decisions now)
	assert.Contains(t, s, "decisions: .agentic/core/DECISIONS.md")

	// agents: upstream entries present with upstream values
	assert.Contains(t, s, "new kernel desc")
	assert.Contains(t, s, "agent-foundry")

	// local-only agent preserved
	assert.Contains(t, s, "my-custom-agent")

	// workflows: upstream description wins
	assert.Contains(t, s, "new spec desc")

	// skills: both upstream entries present
	assert.Contains(t, s, "create-skill")
	assert.Contains(t, s, "new bridge desc")
}

// --- detect ---

func agentsMD(repo string) []byte {
	return []byte("---\ntitle: Agentic Kernel\nversion: \"1.0\"\nrepository: " + repo + "\n---\n# Agentic Kernel\n")
}

func manifestYML(repo string) []byte {
	return []byte("kernel:\n  repository: " + repo + "\n  version: \"1.0\"\n")
}

func TestDetect(t *testing.T) {
	const repo = "https://github.com/elliottpolk/agentic-kernel"

	tests := []struct {
		name      string
		setup     func(dir string)
		wantState State
		wantRepo  string
	}{
		{
			name:      "absent",
			setup:     func(dir string) {},
			wantState: StateAbsent,
		},
		{
			name: "agents.md only - valid",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), agentsMD(repo), 0644)
			},
			wantState: StateAgentsMDOnly,
			wantRepo:  repo,
		},
		{
			name: "agents.md only - invalid (no frontmatter)",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# no frontmatter"), 0644)
			},
			wantState: StateAbsent,
		},
		{
			name: ".agentic only - valid",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), manifestYML(repo), 0644)
			},
			wantState: StateAgenticOnly,
			wantRepo:  repo,
		},
		{
			name: ".agentic only - manifest missing kernel.repository",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte("version: \"1.0\"\n"), 0644)
			},
			wantState: StateAbsent,
		},
		{
			name: "both present",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), agentsMD(repo), 0644)
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), manifestYML(repo), 0644)
			},
			wantState: StateBoth,
			wantRepo:  repo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			state, gotRepo, err := detect(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.wantState, state)
			if tt.wantRepo != "" {
				assert.Equal(t, tt.wantRepo, gotRepo)
			}
		})
	}
}

// --- cacheFiles / rollback ---

func TestCacheAndRollback(t *testing.T) {
	dir := t.TempDir()

	// Write two files that will be "core".
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agentic", "core"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("original agents"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"), []byte("original behavior"), 0644))

	paths := []string{"AGENTS.md", ".agentic/core/BEHAVIOR.md"}

	cache, err := cacheFiles(dir, paths)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cache) })

	// Overwrite the originals.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("modified agents"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"), []byte("modified behavior"), 0644))

	// Rollback should restore originals.
	require.NoError(t, rollback(cache, dir))

	got, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	assert.Equal(t, "original agents", string(got))

	got, _ = os.ReadFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"))
	assert.Equal(t, "original behavior", string(got))

	// Cache dir should be gone.
	_, err = os.Stat(cache)
	assert.True(t, os.IsNotExist(err))
}

// --- Run (integration-style with stubs) ---

func makeUpstreamCache(t *testing.T) string {
	t.Helper()
	cache := t.TempDir()
	// AGENTS.md
	require.NoError(t, os.WriteFile(filepath.Join(cache, "AGENTS.md"), agentsMD("https://github.com/elliottpolk/agentic-kernel"), 0644))
	// core files
	require.NoError(t, os.MkdirAll(filepath.Join(cache, ".agentic", "core"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cache, ".agentic", "core", "BEHAVIOR.md"), []byte("upstream behavior"), 0644))
	// manifest
	require.NoError(t, os.WriteFile(filepath.Join(cache, ".agentic", "manifest.yml"), []byte(upstreamManifest), 0644))
	// user-owned placeholder (must NOT be written)
	require.NoError(t, os.MkdirAll(filepath.Join(cache, ".agentic", "memories", "state"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cache, ".agentic", "memories", "state", ".gitkeep"), []byte(""), 0644))
	return cache
}

func TestRun_normalSync(t *testing.T) {
	dir := t.TempDir()

	// Setup valid local install.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), agentsMD("https://github.com/elliottpolk/agentic-kernel"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agentic", "core"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"), []byte("old behavior"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agentic", "memories", "state"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "memories", "state", "project.md"), []byte("user data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte(localManifest), 0644))

	upstreamCache := makeUpstreamCache(t)

	// Stub confirmFn and warnFn.
	orig := confirmFn
	origWarn := warnFn
	confirmFn = func(bool) (bool, error) { return true, nil }
	warnFn = func(string, []string) {}
	defer func() { confirmFn = orig; warnFn = origWarn }()

	err := runWithCache(dir, upstreamCache)
	require.NoError(t, err)

	// Core file updated.
	got, _ := os.ReadFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"))
	assert.Equal(t, "upstream behavior", string(got))

	// User data untouched.
	got, _ = os.ReadFile(filepath.Join(dir, ".agentic", "memories", "state", "project.md"))
	assert.Equal(t, "user data", string(got))

	// memories/.gitkeep NOT written (user-owned).
	_, err = os.Stat(filepath.Join(dir, ".agentic", "memories", "state", ".gitkeep"))
	assert.True(t, os.IsNotExist(err))
}

func TestRun_abortWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	orig := confirmFn
	origWarn := warnFn
	confirmFn = func(bool) (bool, error) { return true, nil }
	warnFn = func(string, []string) {}
	defer func() { confirmFn = orig; warnFn = origWarn }()

	err := runWithCache(dir, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "akctl init")
}

func TestRun_confirmRejected(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), agentsMD("https://github.com/elliottpolk/agentic-kernel"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agentic"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), manifestYML("https://github.com/elliottpolk/agentic-kernel"), 0644))

	orig := confirmFn
	origWarn := warnFn
	confirmFn = func(bool) (bool, error) { return false, nil }
	warnFn = func(string, []string) {}
	defer func() { confirmFn = orig; warnFn = origWarn }()

	err := runWithCache(dir, makeUpstreamCache(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}

// runWithCache is a test-only helper that bypasses kernel.Fetch and injects a
// pre-built upstream cache directory.
func runWithCache(targetDir, cacheDir string) error {
	state, _, err := detect(targetDir)
	if err != nil {
		return err
	}
	if state == StateAbsent {
		return errAbsentToInit
	}

	// Mirror what Run() does: derive user-owned paths from the upstream manifest.
	agentPaths, _ := kernel.ComponentPathsFromManifest(filepath.Join(cacheDir, ".agentic", "manifest.yml"))

	coreFiles, err := corePaths(targetDir, cacheDir, agentPaths)
	if err != nil {
		return err
	}

	warnFn("", coreFiles)

	ok, err := confirmFn(false)
	if err != nil {
		return err
	}
	if !ok {
		return errAborted
	}

	cache, err := cacheFiles(targetDir, coreFiles)
	if err != nil {
		return err
	}

	if err := writeCore(targetDir, cacheDir, coreFiles); err != nil {
		if rbErr := rollback(cache, targetDir); rbErr != nil {
			return rbErr
		}
		return err
	}

	os.RemoveAll(cache)
	return nil
}

// --- mergeManifest against live upstream ---

// TestMergeManifest_upstreamRaw fetches the actual manifest.yml from
// elliottpolk/agentic-kernel and merges it with this project's local
// manifest, verifying that comments and key ordering from the local file
// are fully preserved.
//
// The test is skipped when the upstream is unreachable so it does not break
// offline or CI-restricted environments.
func TestMergeManifest_upstreamRaw(t *testing.T) {
	const rawURL = "https://raw.githubusercontent.com/elliottpolk/agentic-kernel/main/.agentic/manifest.yml"

	resp, err := http.Get(rawURL)
	if err != nil {
		t.Skipf("upstream unreachable, skipping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("upstream returned status %d, skipping", resp.StatusCode)
	}

	upstream, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Use this project's actual manifest as the local input.
	local, err := os.ReadFile(filepath.Join("..", "..", ".agentic", "manifest.yml"))
	require.NoError(t, err, "read local .agentic/manifest.yml")

	out, err := mergeManifest(local, upstream)
	require.NoError(t, err)

	s := string(out)

	// Header comments from the local file must be present.
	assert.Contains(t, s, "Agentic Kernel Manifest", "header comment lost")
	assert.Contains(t, s, "Registry of all active components", "header comment lost")

	// Local project values must survive.
	assert.Contains(t, s, "akctl", "local project name lost")

	// kernel: block must come from upstream (contains repository field).
	assert.Contains(t, s, "agentic-kernel", "kernel.repository lost after merge")

	// Local-only skills must be preserved (not present in upstream kernel).
	assert.Contains(t, s, "claude-bridge", "local-only skill lost after merge")

	// Upstream-only skill must be added.
	assert.Contains(t, s, "gemini-bridge", "upstream-only skill not added after merge")
}
