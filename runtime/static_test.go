package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestStaticMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Static project",
			path:     "../testdata/static",
			expected: true,
		},
		{
			name:     "Static project with public directory",
			path:     "../testdata/static-public",
			expected: true,
		},
		{
			name:     "Static project with static directory",
			path:     "../testdata/static-static",
			expected: true,
		},
		{
			name:     "Static project with dist directory",
			path:     "../testdata/static-dist",
			expected: true,
		},
		{
			name:     "Not a Static project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			static := &runtime.Static{Log: logger}
			if static.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, static.Match(test.path))
			}
		})
	}
}

func TestStaticGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "Static project",
			path:     "../testdata/static",
			expected: []any{`ARG SERVER_ROOT=.`},
		},
		{
			name:     "Static project with public directory",
			path:     "../testdata/static-public",
			expected: []any{`ARG SERVER_ROOT=public`},
		},
		{
			name:     "Static project with static directory",
			path:     "../testdata/static-static",
			expected: []any{`ARG SERVER_ROOT=static`},
		},
		{
			name:     "Static project with dist directory",
			path:     "../testdata/static-dist",
			expected: []any{`ARG SERVER_ROOT=dist`},
		},
		{
			name:     "Not a Static project",
			path:     "../testdata/deno",
			expected: []any{`ARG SERVER_ROOT=.`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			static := &runtime.Static{Log: logger}
			dockerfile, err := static.GenerateDockerfile(test.path)
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
