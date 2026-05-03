package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	gogithub "github.com/google/go-github/v84/github"
	"golang.org/x/oauth2"
)

// NewClient constructs a GitHub API client. If token is non-empty it creates
// an authenticated client via an oauth2 transport; otherwise it returns an
// unauthenticated client subject to GitHub's lower rate limits.
func NewClient(ctx context.Context, token string) *gogithub.Client {
	if strings.TrimSpace(token) == "" {
		return gogithub.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return gogithub.NewClient(tc)
}

// ResolveToken returns a GitHub token by checking, in order:
//  1. The GITHUB_TOKEN environment variable
//  2. The output of `gh auth token` if gh is installed and authenticated
//
// An error is returned if no token can be found.
func ResolveToken() (string, error) {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t, nil
	}

	if ghPath, err := exec.LookPath("gh"); err == nil {
		out, err := exec.Command(ghPath, "auth", "token").Output()
		if err == nil {
			if t := strings.TrimSpace(string(out)); t != "" {
				return t, nil
			}
		}
	}

	return "", fmt.Errorf(
		"github: no token found — set the GITHUB_TOKEN environment variable " +
			"or authenticate with `gh auth login`",
	)
}
