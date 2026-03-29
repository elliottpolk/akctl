package github

import (
	"context"
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
