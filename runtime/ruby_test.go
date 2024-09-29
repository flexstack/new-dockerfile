package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestRubyMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Ruby project",
			path:     "../testdata/ruby",
			expected: true,
		},
		{
			name:     "Ruby project with config/environment.rb",
			path:     "../testdata/ruby-config-environment",
			expected: true,
		},
		{
			name:     "Ruby project with config.ru",
			path:     "../testdata/ruby-config-ru",
			expected: true,
		},
		{
			name:     "Ruby project with rails",
			path:     "../testdata/ruby-rails",
			expected: true,
		},
		{
			name:     "Not a Ruby project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ruby := &runtime.Ruby{Log: logger}
			if ruby.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, ruby.Match(test.path))
			}
		})
	}
}

func TestRubyGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		data     map[string]string
		expected []any
	}{
		{
			name: "Ruby project",
			path: "../testdata/ruby",
			expected: []any{
				`ARG VERSION=2.0.0`,
				regexp.MustCompile(`^ARG INSTALL_CMD="bundle install"$`),
				regexp.MustCompile(`^ARG BUILD_CMD=$`),
				regexp.MustCompile(`^ARG START_CMD=$`),
			},
		},
		{
			name: "Ruby project w/ mise",
			path: "../testdata/ruby-mise",
			expected: []any{
				`ARG VERSION=2.7`,
				regexp.MustCompile(`^ARG INSTALL_CMD="bundle install"$`),
				regexp.MustCompile(`^ARG BUILD_CMD=$`),
				regexp.MustCompile(`^ARG START_CMD=$`),
			},
		},
		{
			name: "Ruby project with config/environment.rb",
			path: "../testdata/ruby-config-environment",
			expected: []any{
				`ARG VERSION=3.0.1`,
				regexp.MustCompile(`^ARG INSTALL_CMD="bundle install"$`),
				regexp.MustCompile(`^ARG BUILD_CMD=$`),
				`ARG START_CMD="bundle exec ruby script/server"`,
			},
		},
		{
			name: "Ruby project with config.ru",
			path: "../testdata/ruby-config-ru",
			expected: []any{
				`ARG VERSION=2.3.0`,
				regexp.MustCompile(`^ARG INSTALL_CMD="bundle install"$`),
				regexp.MustCompile(`^ARG BUILD_CMD=$`),
				`ARG START_CMD="bundle exec rackup config.ru -p ${PORT}"`,
			},
		},
		{
			name: "Ruby project with build mounts",
			path: "../testdata/ruby-config-ru",
			data: map[string]string{"BuildMounts": `--mount=type=secret,id=_env,target=/app/.env \
    `},
			expected: []any{regexp.MustCompile(`^RUN --mount=type=secret,id=_env,target=/app/.env \\$`)},
		},
		{
			name: "Ruby project with install mounts",
			path: "../testdata/ruby-config-ru",
			data: map[string]string{"InstallMounts": `--mount=type=secret,id=_env,target=/app/.env \
    `},
			expected: []any{regexp.MustCompile(`^RUN --mount=type=secret,id=_env,target=/app/.env \\$`)},
		},
		{
			name: "Ruby project with rails",
			path: "../testdata/ruby-rails",
			expected: []any{
				`ARG VERSION=3.1`,
				`ARG INSTALL_CMD="bundle install && corepack enable pnpm && pnpm i --frozen-lockfile"`,
				`ARG BUILD_CMD="bundle exec rake assets:precompile"`,
				`ARG START_CMD="bundle exec rails server -b 0.0.0.0 -p ${PORT}`,
			},
		},
		{
			name:     "Not a Ruby project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=3.1`, regexp.MustCompile(`^ARG INSTALL_CMD="bundle install"$`), regexp.MustCompile(`^ARG BUILD_CMD=$`), regexp.MustCompile(`^ARG START_CMD=$`)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ruby := &runtime.Ruby{Log: logger}
			dockerfile, err := ruby.GenerateDockerfile(test.path, test.data)
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
