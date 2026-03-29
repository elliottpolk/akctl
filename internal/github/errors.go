package github

import (
	"errors"
	"net/http"

	gogithub "github.com/google/go-github/v84/github"
)

// IsNotFound returns true when err is a GitHub API 404 response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *gogithub.ErrorResponse
	if errors.As(err, &e) {
		return e.Response != nil && e.Response.StatusCode == http.StatusNotFound
	}
	return false
}
