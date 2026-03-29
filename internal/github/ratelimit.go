package github

import (
	"context"
	"fmt"
	"time"

	gogithub "github.com/google/go-github/v84/github"
)

// CheckRateLimit fetches the current GitHub API core rate limit and returns
// an error (including the reset time) if the limit is exhausted. It is
// intended as a pre-flight check before any API calls are made.
func CheckRateLimit(ctx context.Context, client *gogithub.Client) error {
	limits, _, err := client.RateLimit.Get(ctx)
	if err != nil {
		return fmt.Errorf("check rate limit: %w", err)
	}
	if limits.Core != nil && limits.Core.Remaining == 0 {
		return fmt.Errorf("GitHub API rate limit exhausted; resets at %s",
			limits.Core.Reset.Time.Format(time.RFC3339))
	}
	return nil
}
