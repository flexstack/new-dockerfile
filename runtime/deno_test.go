package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestDenoMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Deno project",
			path:     "../testdata/deno",
			expected: true,
		},
		{
			name:     "Deno project with .ts file",
			path:     "../testdata/deno-jsonc",
			expected: true,
		},
		{
			name:     "Not a Deno project",
			path:     "../testdata/ruby",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deno := &runtime.Deno{Log: logger}
			if deno.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, deno.Match(test.path))
			}
		})
	}
}

func TestDenoGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		data     map[string]string
		expected []any
	}{
		{
			name:     "Deno project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=latest`, `ARG INSTALL_CMD="deno cache main.ts"`, `ARG START_CMD="deno run --allow-all main.ts"`},
		},
		{
			name:     "Deno project w/ mise",
			path:     "../testdata/deno-mise",
			expected: []any{`ARG VERSION=1.43.2`, `ARG INSTALL_CMD="deno cache main.ts"`, `ARG START_CMD="deno run --allow-all main.ts"`},
		},
		{
			name:     "Deno project with .ts file",
			path:     "../testdata/deno-jsonc",
			expected: []any{`ARG VERSION=1.43.3`, `ARG INSTALL_CMD="deno task cache"`, `ARG START_CMD="deno task start"`},
		},
		{
			name: "Deno project with install mounts",
			path: "../testdata/deno-jsonc",
			data: map[string]string{"InstallMounts": `--mount=type=secret,id=_env,target=/app/.env \
    `},
			expected: []any{regexp.MustCompile(`^RUN --mount=type=secret,id=_env,target=/app/.env \\$`)},
		},
		{
			name:     "Not a Deno project",
			path:     "../testdata/ruby",
			expected: []any{`ARG VERSION=latest`, regexp.MustCompile(`^ARG INSTALL_CMD=$`), regexp.MustCompile(`^ARG START_CMD=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deno := &runtime.Deno{Log: logger}
			dockerfile, err := deno.GenerateDockerfile(test.path, test.data)
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
