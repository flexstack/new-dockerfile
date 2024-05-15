package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestElixirMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Elixir project",
			path:     "../testdata/elixir",
			expected: true,
		},
		{
			name:     "Elixir project with .tool-versions",
			path:     "../testdata/elixir-tool-versions",
			expected: true,
		},
		{
			name:     "Not a Elixir project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			elixir := &runtime.Elixir{Log: logger}
			if elixir.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, elixir.Match(test.path))
			}
		})
	}
}

func TestElixirGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "Elixir project",
			path:     "../testdata/elixir",
			expected: []any{`ARG VERSION=1.10`, `ARG OTP_VERSION=22`, `ARG BIN_NAME=hello`},
		},
		{
			name:     "Elixir project with .tool-versions",
			path:     "../testdata/elixir-tool-versions",
			expected: []any{`ARG VERSION=1.11`, `ARG OTP_VERSION=23`, `ARG BIN_NAME=hello`},
		},
		{
			name:     "Not a Elixir project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=1.12`, `ARG OTP_VERSION=26`, regexp.MustCompile(`^ARG BIN_NAME=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			elixir := &runtime.Elixir{Log: logger}
			dockerfile, err := elixir.GenerateDockerfile(test.path)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			for _, line := range test.expected {
				found := false
				lines := strings.Split(string(dockerfile), "\n")

				for _, l := range lines {
					switch v := line.(type) {
					case string:
						if strings.Contains(l, v) {
							found = true
							break
						}
					case *regexp.Regexp:
						if v.MatchString(l) {
							found = true
							break
						}
					}
				}

				if !found {
					t.Errorf("expected %v, not found in %v", line, string(dockerfile))
				}
			}
		})
	}
}
