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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/elliottpolk/akctl/internal/gitignore"
	"github.com/elliottpolk/akctl/internal/kernel"
)

// Options controls init behavior.
type Options struct {
	Force     bool
	Debug     bool
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
	// collectMetaFn, confirmFn, and probeMetaFn are package-level vars so tests
	// can inject non-interactive or no-shell implementations without a TTY.
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
		ok, err := confirmFn(opts.Force, paths)
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

	defaults, info := probeMetaFn(target)
	defaults.Name = defaultName

	var debugLines []string
	if opts.Debug {
		debugLines = info.DebugLines()
	}

	meta, err := collectMetaFn(defaults, debugLines)
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

// confirmOverwrite prompts for explicit confirmation unless force is set.
// Safe default is false (do not overwrite).
func confirmOverwrite(force bool, paths []string) (bool, error) {
	if force {
		return true, nil
	}

	var sb strings.Builder
	for _, p := range paths {
		sb.WriteString("  • " + p + "\n")
	}

	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Destructive action").
				Description("The following files will be permanently destroyed:\n\n" + sb.String()),
			huh.NewConfirm().
				Title("This is destructive and cannot be undone. Proceed?").
				Affirmative("Yes").
				Negative("No").
				Value(&ok),
		),
	).WithTheme(huh.ThemeCharm()).WithProgramOptions(tea.WithAltScreen())
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

// collectMeta runs the interactive project metadata form, pre-filling fields
// from defaults where available. When debugLines is non-empty, a 3-line
// bordered debug panel is rendered below the form.
func collectMeta(defaults *projectMeta, debugLines []string) (*projectMeta, error) {
	m := *defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Placeholder(defaults.Name).
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
	).WithTheme(huh.ThemeCharm())

	p := tea.NewProgram(
		metaFormModel{form: form, debugLines: debugLines},
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		return nil, err
	}
	if form.State == huh.StateAborted {
		return nil, huh.ErrUserAborted
	}

	if strings.TrimSpace(m.Name) == "" {
		m.Name = defaults.Name
	}

	return &m, nil
}

// metaFormModel is a BubbleTea model that renders a huh form with an optional
// 3-line debug panel pinned to the bottom of the screen.
type metaFormModel struct {
	form       *huh.Form
	debugLines []string
	width      int
	height     int
}

func (m metaFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m metaFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}

	f, cmd := m.form.Update(msg)
	if form, ok := f.(*huh.Form); ok {
		m.form = form
	}

	if m.form.State != huh.StateNormal {
		return m, tea.Quit
	}
	return m, cmd
}

func (m metaFormModel) View() string {
	if len(m.debugLines) == 0 || m.width == 0 {
		return m.form.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.form.View(),
		renderDebugPanel(m.debugLines, m.width),
	)
}

// renderDebugPanel builds a lipgloss-bordered panel containing exactly 3 lines
// of dimmed diagnostic text, spanning the full terminal width.
func renderDebugPanel(lines []string, width int) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	rows := make([]string, 3)
	for i := range rows {
		if i < len(lines) {
			rows[i] = dimStyle.Render(lines[i])
		}
	}

	// 2 border columns + 2 padding columns = 4 overhead; clamp to safe minimum.
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(innerW).
		Render(strings.Join(rows, "\n"))
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

	if err := gitignore.Ensure(target, gitignore.Pattern); err != nil {
		cleanup(created)
		return fmt.Errorf("update .gitignore: %w", err)
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
