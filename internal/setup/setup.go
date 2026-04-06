// scaffold: generated from docs/specs/init-subcommand.md
package setup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/elliottpolk/akctl/internal/kernel"
	"github.com/elliottpolk/akctl/internal/ui"
)

// Options controls init behavior.
type Options struct {
	Force     bool
	TargetDir string
}

type projectMeta struct {
	Name      string
	Desc      string
	Author    string
	Org       string
	Copyright string
	License   string
	Repo      string
}

var (
	// collectMetaFn and confirmFn are package-level vars so tests can inject
	// non-interactive implementations without a TTY.
	collectMetaFn = collectMeta
	confirmFn     = confirmOverwrite
)

// Run orchestrates the full init sequence.
func Run(k *kernel.KernelInfo, opts Options) error {
	target := opts.TargetDir
	if target == "" {
		target = "."
	}

	agentsmd, dotagentic := checkConflicts(target)

	if agentsmd || dotagentic {
		paths := genDestructList(target, agentsmd, dotagentic)
		showDestructList(paths, true)

		ok, err := confirmFn(opts.Force)
		if err != nil {
			return fmt.Errorf("confirm: %w", err)
		}
		if !ok {
			return fmt.Errorf("aborted")
		}

		if err := destroyConflicts(target, agentsmd, dotagentic); err != nil {
			return fmt.Errorf("remove existing artifacts: %w", err)
		}
	}

	defaultName, err := dirName(target)
	if err != nil {
		return fmt.Errorf("determine project name: %w", err)
	}

	meta, err := collectMetaFn(defaultName)
	if err != nil {
		return fmt.Errorf("collect metadata: %w", err)
	}

	return writeKernel(target, k, meta)
}

// checkConflicts reports whether AGENTS.md or .agentic/ exist in target.
func checkConflicts(target string) (agentsmd, dotagentic bool) {
	if _, err := os.Stat(filepath.Join(target, "AGENTS.md")); err == nil {
		agentsmd = true
	}
	if _, err := os.Stat(filepath.Join(target, ".agentic")); err == nil {
		dotagentic = true
	}
	return
}

// genDestructList builds a sorted list of paths that will be destroyed.
func genDestructList(target string, agentsmd, dotagentic bool) []string {
	var paths []string

	if agentsmd {
		paths = append(paths, filepath.Join(target, "AGENTS.md"))
	}

	if dotagentic {
		_ = filepath.WalkDir(filepath.Join(target, ".agentic"), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
	}

	sort.Strings(paths)
	return paths
}

// showDestructList prints the files that will be destroyed.
// pretty=true renders a tree-like hierarchy; pretty=false prints plain lines.
func showDestructList(paths []string, pretty bool) {
	fmt.Println(ui.WarnStyle.Render("The following will be permanently destroyed:"))

	if !pretty {
		for _, p := range paths {
			fmt.Println(ui.PathStyle.Render("  " + p))
		}
		return
	}

	// Build a simple tree: group by directory, then list files.
	type entry struct {
		dir  string
		file string
	}
	var entries []entry
	for _, p := range paths {
		entries = append(entries, entry{dir: filepath.Dir(p), file: filepath.Base(p)})
	}

	printed := map[string]bool{}
	for i, e := range entries {
		if !printed[e.dir] {
			printed[e.dir] = true
			fmt.Println(ui.PathStyle.Render("  " + e.dir + "/"))
		}
		isLast := i == len(entries)-1 || entries[i+1].dir != e.dir
		branch := "├── "
		if isLast {
			branch = "└── "
		}
		fmt.Println(ui.PathStyle.Render("    " + branch + e.file))
	}
	fmt.Println()
}

// confirmOverwrite prompts for explicit confirmation unless force is set.
// Safe default is false (do not overwrite).
func confirmOverwrite(force bool) (bool, error) {
	if force {
		return true, nil
	}

	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("This is destructive and cannot be undone. Proceed?").
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

// destroyConflicts removes existing AGENTS.md and .agentic/ if present.
func destroyConflicts(target string, agentsmd, dotagentic bool) error {
	if agentsmd {
		if err := os.Remove(filepath.Join(target, "AGENTS.md")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if dotagentic {
		if err := os.RemoveAll(filepath.Join(target, ".agentic")); err != nil {
			return err
		}
	}
	return nil
}

// collectMeta runs the interactive project metadata form.
func collectMeta(defaultName string) (*projectMeta, error) {
	m := &projectMeta{Name: defaultName}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Placeholder(defaultName).
				Value(&m.Name),
			huh.NewInput().
				Title("Description").
				Placeholder("A short description of the project").
				Value(&m.Desc),
			huh.NewInput().
				Title("Author").
				Placeholder("Your name").
				Value(&m.Author),
			huh.NewInput().
				Title("Organization").
				Placeholder("Your org or team name").
				Value(&m.Org),
			huh.NewInput().
				Title("Copyright").
				Placeholder("e.g. © 2026 Your Name").
				Value(&m.Copyright),
			huh.NewInput().
				Title("License").
				Placeholder("e.g. MIT").
				Value(&m.License),
			huh.NewInput().
				Title("Repository URL").
				Placeholder("https://github.com/owner/repo").
				Value(&m.Repo),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(m.Name) == "" {
		m.Name = defaultName
	}

	return m, nil
}

// writeKernel copies files from k.CacheDir to target, injecting project
// metadata into manifest.yml. Tracks created paths for cleanup on error.
func writeKernel(target string, k *kernel.KernelInfo, meta *projectMeta) error {
	var created []string

	writeFile := func(dest string, content []byte) error {
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		created = append(created, dest)
		return nil
	}

	// Write AGENTS.md first.
	agentsSrc := filepath.Join(k.CacheDir, "AGENTS.md")
	agentsContent, err := os.ReadFile(agentsSrc)
	if err != nil {
		cleanup(created)
		return fmt.Errorf("read cached AGENTS.md: %w", err)
	}
	if err := writeFile(filepath.Join(target, "AGENTS.md"), agentsContent); err != nil {
		cleanup(created)
		return err
	}

	// Walk the .agentic/ subtree in the cache.
	cacheAgentic := filepath.Join(k.CacheDir, ".agentic")
	err = filepath.WalkDir(cacheAgentic, func(src string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(k.CacheDir, src)
		dest := filepath.Join(target, rel)

		// User-owned paths (memories) must not be seeded with kernel content,
		// but the directories still need to exist in the target project.
		if strings.HasPrefix(filepath.ToSlash(rel), ".agentic/memories/") {
			return os.MkdirAll(filepath.Dir(dest), 0755)
		}

		content, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read cached %s: %w", rel, err)
		}

		// Inject project metadata into manifest.yml.
		if rel == filepath.Join(".agentic", "manifest.yml") {
			content = injectMeta(content, meta, k)
		}

		return writeFile(dest, content)
	})

	if err != nil {
		cleanup(created)
		return fmt.Errorf("write kernel files: %w", err)
	}

	return nil
}

// injectMeta replaces empty project: field values in the upstream manifest.yml
// with the user-supplied metadata. The kernel: block is left untouched.
func injectMeta(content []byte, meta *projectMeta, k *kernel.KernelInfo) []byte {
	s := string(content)

	replacements := map[string]string{
		`name: ""`:         fmt.Sprintf(`name: "%s"`, meta.Name),
		`description: ""`:  fmt.Sprintf(`description: "%s"`, meta.Desc),
		`author: ""`:       fmt.Sprintf(`author: "%s"`, meta.Author),
		`organization: ""`: fmt.Sprintf(`organization: "%s"`, meta.Org),
		`copyright: ""`:    fmt.Sprintf(`copyright: "%s"`, meta.Copyright),
		`license: ""`:      fmt.Sprintf(`license: "%s"`, meta.License),
		`repository: ""`:   fmt.Sprintf(`repository: "%s"`, meta.Repo),
	}

	for old, new := range replacements {
		s = strings.Replace(s, old, new, 1)
	}

	return []byte(s)
}

// cleanup removes files created during a failed write, in reverse order,
// then prunes any empty directories.
func cleanup(created []string) {
	for i := len(created) - 1; i >= 0; i-- {
		os.Remove(created[i])
	}
	// Prune empty dirs (best effort).
	dirs := map[string]struct{}{}
	for _, p := range created {
		dirs[filepath.Dir(p)] = struct{}{}
	}
	for d := range dirs {
		os.Remove(d) // only removes if empty
	}
}

// dirName returns the kebab-case name of the target directory.
func dirName(target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return toKebab(filepath.Base(abs)), nil
}

// toKebab converts a string to kebab-case.
var nonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func toKebab(s string) string {
	s = strings.ToLower(s)
	s = nonAlpha.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
