package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestBunMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Bun project",
			path:     "../testdata/bun",
			expected: true,
		},
		{
			name:     "Bun project with .ts file",
			path:     "../testdata/bun-bunfig",
			expected: true,
		},
		{
			name:     "Not a Bun project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bun := &runtime.Bun{Log: logger}
			if bun.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, bun.Match(test.path))
			}
		})
	}
}

func TestBunGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		data     map[string]string
		expected []any
	}{
		{
			name:     "Bun project",
			path:     "../testdata/bun",
			expected: []any{`ARG VERSION=1`, `ARG INSTALL_CMD="bun install"`, regexp.MustCompile(`^ARG BUILD_CMD=$`), `ARG START_CMD="bun index.ts"`},
		},
		{
			name:     "Bun project with .ts file",
			path:     "../testdata/bun-bunfig",
			expected: []any{`ARG VERSION=1.1.4`, `ARG INSTALL_CMD="bun install"`, `ARG BUILD_CMD="bun run build:prod"`, `ARG START_CMD="bun run start:production"`},
		},
		{
			name: "Bun project with build mounts",
			path: "../testdata/bun-bunfig",
			data: map[string]string{"BuildMounts": `--mount=type=secret,id=_env,target=/app/.env \
    `},
			expected: []any{regexp.MustCompile(`^RUN --mount=type=secret,id=_env,target=/app/.env \\$`)},
		},
		{
			name: "Bun project with install mounts",
			path: "../testdata/bun-bunfig",
			data: map[string]string{"InstallMounts": `--mount=type=secret,id=_env,target=/app/.env \
    `},
			expected: []any{regexp.MustCompile(`^RUN --mount=type=secret,id=_env,target=/app/.env \\$`)},
		},
		{
			name:     "Not a Bun project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=1`, regexp.MustCompile(`^ARG INSTALL_CMD="bun install"`), regexp.MustCompile(`^ARG BUILD_CMD=$`), regexp.MustCompile(`^ARG START_CMD=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bun := &runtime.Bun{Log: logger}
			dockerfile, err := bun.GenerateDockerfile(test.path, test.data)
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
