package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestRustMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Rust project",
			path:     "../testdata/rust",
			expected: true,
		},
		{
			name:     "Rust project with  [[bin]] directive",
			path:     "../testdata/rust-bin",
			expected: true,
		},
		{
			name:     "Not a Rust project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rust := &runtime.Rust{Log: logger}
			if rust.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, rust.Match(test.path))
			}
		})
	}
}

func TestRustGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "Rust project",
			path:     "../testdata/rust",
			expected: []any{`ARG BIN_NAME=ingest`},
		},
		{
			name:     "Rust project with [[bin]] directive",
			path:     "../testdata/rust-bin",
			expected: []any{`ARG BIN_NAME=rg`},
		},
		{
			name:     "Not a Rust project",
			path:     "../testdata/deno",
			expected: []any{regexp.MustCompile(`^ARG BIN_NAME=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rust := &runtime.Rust{Log: logger}
			dockerfile, err := rust.GenerateDockerfile(test.path)
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
