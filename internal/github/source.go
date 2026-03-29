package github

import (
	"fmt"
	"strings"
)

// DefaultSource is the canonical kernel used when --kernel.source is omitted.
const DefaultSource = "github.com/elliottpolk/agentic-kernel"

// ParseSource parses a kernel source string of the form "host/owner/repo".
// An empty or whitespace-only value resolves to DefaultSource.
// Any host other than "github.com" returns an error.
func ParseSource(source string) (owner, repo string, err error) {
	src := strings.TrimSpace(source)
	if src == "" {
		src = DefaultSource
	}

	parts := strings.SplitN(src, "/", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid kernel source %q: expected github.com/<owner>/<repo>", src)
	}

	host := parts[0]
	if host != "github.com" {
		return "", "", fmt.Errorf("provider %q is not yet supported; only github.com is supported", host)
	}

	return parts[1], parts[2], nil
}
