package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestGolangMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Golang project",
			path:     "../testdata/go",
			expected: true,
		},
		{
			name:     "Golang project with go.mod file",
			path:     "../testdata/go-mod",
			expected: true,
		},
		{
			name:     "Not a Golang project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			golang := &runtime.Golang{Log: logger}
			if golang.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, golang.Match(test.path))
			}
		})
	}
}

func TestGolangGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "Golang project",
			path:     "../testdata/go",
			expected: []any{`ARG VERSION=1.16.3`, `ARG PACKAGE=./main.go`},
		},
		{
			name:     "Golang project w/ mise",
			path:     "../testdata/go-mise",
			expected: []any{`ARG VERSION=1.16`, `ARG PACKAGE=./main.go`},
		},
		{
			name:     "Golang project with go.mod file",
			path:     "../testdata/go-mod",
			expected: []any{`ARG VERSION=1.22.3`, `ARG PACKAGE=./cmd/hello`},
		},
		{
			name:     "Not a Golang project",
			path:     "../testdata/ruby",
			expected: []any{`ARG VERSION=1.17`, regexp.MustCompile(`^ARG PACKAGE=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			golang := &runtime.Golang{Log: logger}
			dockerfile, err := golang.GenerateDockerfile(test.path)
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
