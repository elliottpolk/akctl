package github

import (
	"fmt"
	"os/exec"
	"runtime"
)

const ghInstallURL = "https://cli.github.com/"

// EnsureGH verifies that the gh CLI is available in PATH. If it is not found,
// it attempts a platform-specific installation and re-checks. Returns an error
// if gh cannot be found or installed, directing the user to ghInstallURL.
func EnsureGH() error {
	if _, err := exec.LookPath("gh"); err == nil {
		return nil
	}

	if err := installGH(); err != nil {
		return fmt.Errorf(
			"gh CLI not found and automatic installation failed (%w); "+
				"please install it manually from %s",
			err, ghInstallURL,
		)
	}

	// Re-verify after the install attempt.
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf(
			"gh CLI was not found after installation attempt; "+
				"please install it manually from %s",
			ghInstallURL,
		)
	}

	return nil
}

// installGH attempts to install the gh CLI using the platform's package manager.
func installGH() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("winget", "install", "--id", "GitHub.cli")
	case "darwin":
		cmd = exec.Command("brew", "install", "gh")
	case "linux":
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "gh")
	default:
		return fmt.Errorf("unsupported OS %q; install gh from %s", runtime.GOOS, ghInstallURL)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install command failed: %w", err)
	}

	return nil
}
