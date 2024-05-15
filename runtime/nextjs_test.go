package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestNextJSMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "NextJS project",
			path:     "../testdata/nextjs",
			expected: true,
		},
		{
			name:     "NextJS project with standalone output",
			path:     "../testdata/nextjs-standalone",
			expected: true,
		},
		{
			name:     "Not a NextJS project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextjs := &runtime.NextJS{Log: logger}
			if nextjs.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, nextjs.Match(test.path))
			}
		})
	}
}

func TestNextJSGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "NextJS project",
			path:     "../testdata/nextjs",
			expected: []any{`ARG VERSION=lts`, `CMD ["node_modules/.bin/next", "start", "-H", "0.0.0.0"]`},
		},
		{
			name:     "NextJS project with standalone output",
			path:     "../testdata/nextjs-standalone",
			expected: []any{`ARG VERSION=16.0.0`, `CMD HOSTNAME="0.0.0.0" node server.js`},
		},
		{
			name:     "Not a NextJS project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=lts`, `CMD ["node_modules/.bin/next", "start", "-H", "0.0.0.0"]`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextjs := &runtime.NextJS{Log: logger}
			dockerfile, err := nextjs.GenerateDockerfile(test.path)
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
