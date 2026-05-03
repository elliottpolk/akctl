package setup

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// probeMetaFn is a package-level var so tests can inject a non-shell implementation.
var probeMetaFn = probeRepoMeta

// ghRepoView is the shape of `gh repo view --json description,owner,licenseInfo,url`.
type ghRepoView struct {
	Description string `json:"description"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	LicenseInfo struct {
		SpdxID string `json:"spdxId"`
		Name   string `json:"name"`
	} `json:"licenseInfo"`
	URL string `json:"url"`
}

// probeInfo carries diagnostic information from a probeRepoMeta call.
// DebugLines returns exactly 3 strings suitable for display in the TUI.
type probeInfo struct {
	ghPath string // resolved path to gh binary; empty if not found
	cmdRun string // the command string that was attempted
	errMsg string // error from cmd.Output() or JSON parse, empty on success
	meta   *projectMeta
}

func (p probeInfo) DebugLines() []string {
	line1 := "gh:     not found in PATH"
	if p.ghPath != "" {
		line1 = "gh:     " + p.ghPath
	}

	line2 := "cmd:    (skipped)"
	if p.cmdRun != "" {
		line2 = "cmd:    " + p.cmdRun
	}

	var line3 string
	switch {
	case p.errMsg != "":
		line3 = "error:  " + p.errMsg
	case p.meta != nil:
		found := func(s string) string {
			if strings.TrimSpace(s) != "" {
				return "✓"
			}
			return "✗"
		}
		line3 = fmt.Sprintf(
			"result: desc=%s  org=%s  license=%s  repo=%s",
			found(p.meta.Desc), found(p.meta.Org),
			found(p.meta.License), found(p.meta.Repo),
		)
	default:
		line3 = "result: (none)"
	}

	return []string{line1, line2, line3}
}

// probeRepoMeta attempts to derive project metadata from the target directory's
// GitHub remote using the gh CLI. Any field that cannot be determined is left
// empty. Errors are captured in probeInfo for optional debug display; they never
// block init.
func probeRepoMeta(dir string) (*projectMeta, probeInfo) {
	const jsonFields = "description,owner,licenseInfo,url"

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return &projectMeta{}, probeInfo{}
	}

	cmdStr := "gh repo view --json " + jsonFields
	cmd := exec.Command(ghPath, "repo", "view", "--json", jsonFields)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		return &projectMeta{}, probeInfo{ghPath: ghPath, cmdRun: cmdStr, errMsg: msg}
	}

	var v ghRepoView
	if err := json.Unmarshal(out, &v); err != nil {
		return &projectMeta{}, probeInfo{ghPath: ghPath, cmdRun: cmdStr, errMsg: "parse: " + err.Error()}
	}

	license := strings.TrimSpace(v.LicenseInfo.SpdxID)
	if license == "" {
		license = strings.TrimSpace(v.LicenseInfo.Name)
	}

	m := &projectMeta{
		Desc:    strings.TrimSpace(v.Description),
		Org:     strings.TrimSpace(v.Owner.Login),
		License: license,
		Repo:    strings.TrimSpace(v.URL),
	}
	return m, probeInfo{ghPath: ghPath, cmdRun: cmdStr, meta: m}
}
