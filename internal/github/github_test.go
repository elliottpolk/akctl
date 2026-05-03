package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghpkg "github.com/elliottpolk/akctl/internal/github"
)

// newGHTestClient returns a go-github client pointed at the given test server.
func newGHTestClient(t *testing.T, srv *httptest.Server) *gogithub.Client {
	t.Helper()
	client := gogithub.NewClient(nil)
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = base
	client.UploadURL = base
	return client
}

// --- ParseSource ---

func TestParseSource(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "valid source",
			input:     "github.com/elliottpolk/agentic-kernel",
			wantOwner: "elliottpolk",
			wantRepo:  "agentic-kernel",
		},
		{
			name:      "empty resolves to default",
			input:     "",
			wantOwner: "elliottpolk",
			wantRepo:  "agentic-kernel",
		},
		{
			name:      "whitespace resolves to default",
			input:     "   ",
			wantOwner: "elliottpolk",
			wantRepo:  "agentic-kernel",
		},
		{
			name:    "non-github host rejected",
			input:   "gitlab.com/foo/bar",
			wantErr: true,
		},
		{
			name:    "missing repo segment rejected",
			input:   "github.com/owner",
			wantErr: true,
		},
		{
			name:    "no slashes rejected",
			input:   "notahost",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ghpkg.ParseSource(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

// --- CheckRateLimit ---

// rateLimitBody builds a minimal /rate_limit response body.
func rateLimitBody(remaining int, reset time.Time) map[string]any {
	return map[string]any{
		"resources": map[string]any{
			"core": map[string]any{
				"limit":     5000,
				"remaining": remaining,
				"reset":     reset.Unix(),
			},
		},
		"rate": map[string]any{
			"limit":     5000,
			"remaining": remaining,
			"reset":     reset.Unix(),
		},
	}
}

func TestCheckRateLimit(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "remaining > 0 returns nil",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(rateLimitBody(100, time.Now().Add(time.Hour)))
			},
		},
		{
			name: "remaining == 0 returns error with reset time",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(rateLimitBody(0, time.Now().Add(30*time.Minute)))
			},
			wantErr: true,
		},
		{
			name: "API error returns wrapped error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server error", http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := newGHTestClient(t, srv)
			err := ghpkg.CheckRateLimit(context.Background(), client)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// --- IsNotFound ---

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "404 ErrorResponse returns true",
			err: &gogithub.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusNotFound},
			},
			want: true,
		},
		{
			name: "403 ErrorResponse returns false",
			err: &gogithub.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusForbidden},
			},
			want: false,
		},
		{
			name: "non-github error returns false",
			err:  assert.AnError,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ghpkg.IsNotFound(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ResolveToken ---

func TestResolveToken(t *testing.T) {
	t.Run("GITHUB_TOKEN env var is used", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "env-token-value")

		token, err := ghpkg.ResolveToken()
		require.NoError(t, err)
		assert.Equal(t, "env-token-value", token)
	})

	t.Run("returns error when no token source available", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")

		// This test will succeed when gh is not authenticated (gh auth token returns empty/error)
		// or gh is not installed. We can only reliably assert the shape of the error.
		_, err := ghpkg.ResolveToken()

		// In CI or environments without gh auth, this should return an error.
		// In environments where gh is authenticated, the test is skipped to avoid
		// false failures due to a valid gh session.
		if err == nil {
			t.Skip("gh CLI is authenticated in this environment; skipping no-token error test")
		}
		assert.ErrorContains(t, err, "no token found")
	})
}
