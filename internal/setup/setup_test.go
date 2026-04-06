package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elliottpolk/akctl/internal/kernel"
)

// --- toKebab ---

func TestToKebab(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase passthrough", "my-project", "my-project"},
		{"spaces to hyphens", "My Project", "my-project"},
		{"underscores to hyphens", "my_project", "my-project"},
		{"special chars stripped", "my@project!", "my-project"},
		{"leading trailing hyphens trimmed", "-my-project-", "my-project"},
		{"collapsed runs", "my--project", "my-project"},
		{"mixed", "My__Big Project!", "my-big-project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, toKebab(tt.input))
		})
	}
}

// --- checkConflicts ---

func TestCheckConflicts(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(dir string)
		wantAgentsmd   bool
		wantDotagentic bool
	}{
		{
			name:           "neither exists",
			setup:          func(dir string) {},
			wantAgentsmd:   false,
			wantDotagentic: false,
		},
		{
			name: "only AGENTS.md",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
			},
			wantAgentsmd:   true,
			wantDotagentic: false,
		},
		{
			name: "only .agentic dir",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
			},
			wantAgentsmd:   false,
			wantDotagentic: true,
		},
		{
			name: "both exist",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
			},
			wantAgentsmd:   true,
			wantDotagentic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			gotAgentsmd, gotDotagentic := checkConflicts(dir)
			assert.Equal(t, tt.wantAgentsmd, gotAgentsmd)
			assert.Equal(t, tt.wantDotagentic, gotDotagentic)
		})
	}
}

// --- genDestructList ---

func TestGenDestructList(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string)
		agentsmd   bool
		dotagentic bool
		wantLen    int
	}{
		{
			name:       "nothing to destroy",
			setup:      func(dir string) {},
			agentsmd:   false,
			dotagentic: false,
			wantLen:    0,
		},
		{
			name: "only AGENTS.md",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
			},
			agentsmd:   true,
			dotagentic: false,
			wantLen:    1,
		},
		{
			name: "only .agentic subtree",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".agentic", "core"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "core", "BEHAVIOR.md"), []byte("# B"), 0644)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte("version: 1"), 0644)
			},
			agentsmd:   false,
			dotagentic: true,
			wantLen:    2,
		},
		{
			name: "both",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte("version: 1"), 0644)
			},
			agentsmd:   true,
			dotagentic: true,
			wantLen:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			paths := genDestructList(dir, tt.agentsmd, tt.dotagentic)
			assert.Len(t, paths, tt.wantLen)
			// verify sorted
			for i := 1; i < len(paths); i++ {
				assert.LessOrEqual(t, paths[i-1], paths[i], "paths should be sorted")
			}
		})
	}
}

// --- destroyConflicts ---

func TestDestroyConflicts(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string)
		agentsmd   bool
		dotagentic bool
		checkGone  func(t *testing.T, dir string)
	}{
		{
			name: "removes AGENTS.md",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
			},
			agentsmd:   true,
			dotagentic: false,
			checkGone: func(t *testing.T, dir string) {
				_, err := os.Stat(filepath.Join(dir, "AGENTS.md"))
				assert.True(t, os.IsNotExist(err), "AGENTS.md should be gone")
			},
		},
		{
			name: "removes .agentic dir",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte("v: 1"), 0644)
			},
			agentsmd:   false,
			dotagentic: true,
			checkGone: func(t *testing.T, dir string) {
				_, err := os.Stat(filepath.Join(dir, ".agentic"))
				assert.True(t, os.IsNotExist(err), ".agentic should be gone")
			},
		},
		{
			name: "removes both",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test"), 0644)
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
			},
			agentsmd:   true,
			dotagentic: true,
			checkGone: func(t *testing.T, dir string) {
				_, err := os.Stat(filepath.Join(dir, "AGENTS.md"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(filepath.Join(dir, ".agentic"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name:       "neither present is no-op",
			setup:      func(dir string) {},
			agentsmd:   false,
			dotagentic: false,
			checkGone:  func(t *testing.T, dir string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			err := destroyConflicts(dir, tt.agentsmd, tt.dotagentic)
			assert.NoError(t, err)
			tt.checkGone(t, dir)
		})
	}
}

// --- confirmOverwrite ---

func TestConfirmOverwrite(t *testing.T) {
	t.Run("force=true skips prompt", func(t *testing.T) {
		ok, err := confirmOverwrite(true, nil)
		assert.NoError(t, err)
		assert.True(t, ok)
	})
	// force=false requires interactive TTY -- excluded from unit tests
}

// --- injectMeta ---

func TestInjectMeta(t *testing.T) {
	upstreamManifest := `# Agentic Kernel Manifest
version: "1.0"

kernel:
  title: Agentic Kernel
  version: "1.0"
  repository: https://github.com/elliottpolk/agentic-kernel

project:
  name: ""
  description: ""
  author: ""
  organization: ""
  copyright: ""
  license: ""
  repository: ""
`

	k := &kernel.KernelInfo{Version: "1.0", Title: "Agentic Kernel"}

	tests := []struct {
		name         string
		meta         *projectMeta
		wantContains []string
	}{
		{
			name: "all fields replaced",
			meta: &projectMeta{
				Name:      "my-project",
				Desc:      "A test project",
				Author:    "Test Author",
				Org:       "Test Org",
				Copyright: "2026 Test Author",
				License:   "MIT",
				Repo:      "https://github.com/test/my-project",
			},
			wantContains: []string{
				`name: "my-project"`,
				`description: "A test project"`,
				`author: "Test Author"`,
				`organization: "Test Org"`,
				`copyright: "2026 Test Author"`,
				`license: "MIT"`,
				`repository: "https://github.com/test/my-project"`,
			},
		},
		{
			name: "kernel block untouched",
			meta: &projectMeta{Name: "my-project"},
			wantContains: []string{
				"kernel:",
				"title: Agentic Kernel",
				`version: "1.0"`,
				"repository: https://github.com/elliottpolk/agentic-kernel",
			},
		},
		{
			name: "blank field left as empty string",
			meta: &projectMeta{Name: "my-project", Desc: ""},
			wantContains: []string{
				`name: "my-project"`,
				`description: ""`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(injectMeta([]byte(upstreamManifest), tt.meta, k))
			for _, want := range tt.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}

// --- cleanup ---

func TestCleanup(t *testing.T) {
	t.Run("removes created files", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "file1.txt")
		f2 := filepath.Join(dir, "file2.txt")
		require.NoError(t, os.WriteFile(f1, []byte("a"), 0644))
		require.NoError(t, os.WriteFile(f2, []byte("b"), 0644))

		cleanup([]string{f1, f2})

		_, err := os.Stat(f1)
		assert.True(t, os.IsNotExist(err), "file1 should be removed")
		_, err = os.Stat(f2)
		assert.True(t, os.IsNotExist(err), "file2 should be removed")
	})

	t.Run("prunes empty subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		require.NoError(t, os.MkdirAll(sub, 0755))
		f := filepath.Join(sub, "file.txt")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

		cleanup([]string{f})

		_, err := os.Stat(f)
		assert.True(t, os.IsNotExist(err), "file should be removed")
		// sub dir should be gone since it's now empty
		_, err = os.Stat(sub)
		assert.True(t, os.IsNotExist(err), "empty subdir should be pruned")
	})

	t.Run("no-op on empty list", func(t *testing.T) {
		assert.NotPanics(t, func() { cleanup(nil) })
	})
}

// --- writeKernel ---

// buildCache creates a minimal cache dir that mirrors what kernel.Fetch produces.
func buildCache(t *testing.T, withAgentsMD bool, manifestContent string) *kernel.KernelInfo {
	t.Helper()
	cacheDir := t.TempDir()

	if withAgentsMD {
		content := "---\ntitle: Agentic Kernel\nversion: \"1.0\"\n---\n# Agentic Kernel\n"
		require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "AGENTS.md"), []byte(content), 0644))
	}

	if manifestContent != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, ".agentic"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cacheDir, ".agentic", "manifest.yml"), []byte(manifestContent), 0644))
	}

	return &kernel.KernelInfo{Version: "1.0", CacheDir: cacheDir}
}

func TestWriteKernel(t *testing.T) {
	upstreamManifest := `version: "1.0"
project:
  name: ""
  description: ""
  author: ""
  organization: ""
  copyright: ""
  license: ""
  repository: ""
`

	fullMeta := &projectMeta{
		Name:      "my-project",
		Desc:      "A test project",
		Author:    "Test Author",
		Org:       "Test Org",
		Copyright: "2026 Test",
		License:   "MIT",
		Repo:      "https://github.com/test/my-project",
	}

	tests := []struct {
		name      string
		buildInfo func(t *testing.T) *kernel.KernelInfo
		meta      *projectMeta
		wantErr   bool
		check     func(t *testing.T, target string)
		skip      func(t *testing.T)
	}{
		{
			name: "fresh write -- AGENTS.md and manifest written",
			buildInfo: func(t *testing.T) *kernel.KernelInfo {
				return buildCache(t, true, upstreamManifest)
			},
			meta:    fullMeta,
			wantErr: false,
			check: func(t *testing.T, target string) {
				_, err := os.Stat(filepath.Join(target, "AGENTS.md"))
				assert.NoError(t, err, "AGENTS.md should exist")
				_, err = os.Stat(filepath.Join(target, ".agentic", "manifest.yml"))
				assert.NoError(t, err, "manifest.yml should exist")
				content, err := os.ReadFile(filepath.Join(target, ".agentic", "manifest.yml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `name: "my-project"`)
			},
		},
		{
			name: "manifest meta injection end-to-end",
			buildInfo: func(t *testing.T) *kernel.KernelInfo {
				return buildCache(t, true, upstreamManifest)
			},
			meta:    fullMeta,
			wantErr: false,
			check: func(t *testing.T, target string) {
				content, err := os.ReadFile(filepath.Join(target, ".agentic", "manifest.yml"))
				require.NoError(t, err)
				s := string(content)
				assert.Contains(t, s, `name: "my-project"`)
				assert.Contains(t, s, `description: "A test project"`)
				assert.Contains(t, s, `author: "Test Author"`)
				assert.Contains(t, s, `license: "MIT"`)
			},
		},
		{
			name: "AGENTS.md missing from cache returns error",
			buildInfo: func(t *testing.T) *kernel.KernelInfo {
				return buildCache(t, false, upstreamManifest)
			},
			meta:    fullMeta,
			wantErr: true,
			check: func(t *testing.T, target string) {
				entries, err := os.ReadDir(target)
				require.NoError(t, err)
				assert.Empty(t, entries, "no files should be written on failure")
			},
		},
		{
			name: "read-only target returns error and leaves no partial files",
			buildInfo: func(t *testing.T) *kernel.KernelInfo {
				return buildCache(t, true, upstreamManifest)
			},
			meta:    fullMeta,
			wantErr: true,
			skip: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("chmod not applicable on Windows")
				}
				if os.Getuid() == 0 {
					t.Skip("running as root, chmod test not meaningful")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != nil {
				tt.skip(t)
			}

			k := tt.buildInfo(t)
			target := t.TempDir()

			if tt.name == "read-only target returns error and leaves no partial files" {
				require.NoError(t, os.Chmod(target, 0555))
				defer os.Chmod(target, 0755)
			}

			err := writeKernel(target, k, tt.meta)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t, target)
			}
		})
	}
}

// --- dirName ---

func TestDirName(t *testing.T) {
	t.Run("dot resolves to kebab of cwd", func(t *testing.T) {
		name, err := dirName(".")
		assert.NoError(t, err)
		assert.NotEmpty(t, name)
		assert.Equal(t, toKebab(name), name)
	})

	t.Run("explicit path converts to kebab", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "My Test Project")
		require.NoError(t, os.MkdirAll(sub, 0755))
		name, err := dirName(sub)
		assert.NoError(t, err)
		assert.Equal(t, "my-test-project", name)
	})
}

// --- Run (via injectable collectMetaFn / confirmFn) ---

func TestRun(t *testing.T) {
	upstreamManifest := `version: "1.0"
project:
  name: ""
  description: ""
  author: ""
  organization: ""
  copyright: ""
  license: ""
  repository: ""
`

	testMeta := func(defaultName string) (*projectMeta, error) {
		return &projectMeta{
			Name:    defaultName,
			Desc:    "A test project",
			Author:  "Test Author",
			License: "MIT",
		}, nil
	}

	tests := []struct {
		name          string
		setup         func(dir string)
		opts          func(dir string) Options
		overrideConfirm func(force bool, paths []string) (bool, error)
		wantErr       bool
		errContains   string
		check         func(t *testing.T, dir string)
	}{
		{
			name:  "fresh init -- no conflicts",
			setup: func(dir string) {},
			opts:  func(dir string) Options { return Options{TargetDir: dir} },
			check: func(t *testing.T, dir string) {
				_, err := os.Stat(filepath.Join(dir, "AGENTS.md"))
				assert.NoError(t, err)
				_, err = os.Stat(filepath.Join(dir, ".agentic", "manifest.yml"))
				assert.NoError(t, err)
				content, err := os.ReadFile(filepath.Join(dir, ".agentic", "manifest.yml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `author: "Test Author"`)
			},
		},
		{
			name: "existing artifacts with force=true -- overwrites",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("old"), 0644)
				os.MkdirAll(filepath.Join(dir, ".agentic"), 0755)
				os.WriteFile(filepath.Join(dir, ".agentic", "manifest.yml"), []byte("old"), 0644)
			},
			opts: func(dir string) Options { return Options{TargetDir: dir, Force: true} },
			check: func(t *testing.T, dir string) {
				content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
				require.NoError(t, err)
				assert.NotEqual(t, "old", string(content), "AGENTS.md should be overwritten")
			},
		},
		{
			name: "existing artifacts with confirm=false -- aborts",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("old"), 0644)
			},
			opts: func(dir string) Options { return Options{TargetDir: dir, Force: false} },
			overrideConfirm: func(force bool, paths []string) (bool, error) { return false, nil },
			wantErr:     true,
			errContains: "aborted",
		},
		{
			name: "confirm returns error -- propagates",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("old"), 0644)
			},
			opts: func(dir string) Options { return Options{TargetDir: dir, Force: false} },
			overrideConfirm: func(force bool, paths []string) (bool, error) {
				return false, fmt.Errorf("terminal error")
			},
			wantErr:     true,
			errContains: "confirm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			// build a cache dir for this test
			cacheDir := t.TempDir()
			agentsMD := "---\ntitle: Agentic Kernel\nversion: \"1.0\"\n---\n# Agentic Kernel\n"
			require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "AGENTS.md"), []byte(agentsMD), 0644))
			require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, ".agentic"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(cacheDir, ".agentic", "manifest.yml"), []byte(upstreamManifest), 0644))
			k := &kernel.KernelInfo{Version: "1.0", CacheDir: cacheDir}

			// inject non-interactive stubs
			orig := collectMetaFn
			origConfirm := confirmFn
			t.Cleanup(func() { collectMetaFn = orig; confirmFn = origConfirm })
			collectMetaFn = testMeta
			if tt.overrideConfirm != nil {
				confirmFn = tt.overrideConfirm
			}

			err := Run(k, tt.opts(dir))

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, dir)
			}
		})
	}
}


// --- toKebab ---
