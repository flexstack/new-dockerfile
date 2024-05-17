package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestNodeMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Node project",
			path:     "../testdata/node",
			expected: true,
		},
		{
			name:     "Node project with pnpm",
			path:     "../testdata/node-pnpm",
			expected: true,
		},
		{
			name:     "Node project with yarn",
			path:     "../testdata/node-yarn",
			expected: true,
		},
		{
			name:     "Not a Node project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := &runtime.Node{Log: logger}
			if node.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, node.Match(test.path))
			}
		})
	}
}

func TestNodeGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "Node project",
			path:     "../testdata/node",
			expected: []any{`ARG VERSION=lts`, `ARG INSTALL_CMD="npm ci"`, regexp.MustCompile(`^ARG BUILD_CMD=$`), `ARG START_CMD="node index.ts"`},
		},
		{
			name:     "Node project with pnpm",
			path:     "../testdata/node-pnpm",
			expected: []any{`ARG VERSION=16.0.0`, `ARG INSTALL_CMD="corepack enable pnpm && pnpm i --frozen-lockfile"`, `ARG BUILD_CMD="pnpm run build:prod"`, `ARG START_CMD="pnpm run start:production"`},
		},
		{
			name:     "Node project with yarn",
			path:     "../testdata/node-yarn",
			expected: []any{`ARG VERSION=16.0.0`, `ARG INSTALL_CMD="yarn --frozen-lockfile"`, `ARG BUILD_CMD="yarn run build:prod"`, `ARG START_CMD="yarn run start-it"`},
		},
		{
			name:     "Not a Node project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=lts`, regexp.MustCompile(`^ARG INSTALL_CMD="npm ci"`), regexp.MustCompile(`^ARG BUILD_CMD=$`), regexp.MustCompile(`^ARG START_CMD=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := &runtime.Node{Log: logger}
			dockerfile, err := node.GenerateDockerfile(test.path)
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
