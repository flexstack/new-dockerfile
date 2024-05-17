package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestPHPMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "PHP project",
			path:     "../testdata/php",
			expected: true,
		},
		{
			name:     "PHP project with composer",
			path:     "../testdata/php-composer",
			expected: true,
		},
		{
			name:     "PHP project with NPM",
			path:     "../testdata/php-npm",
			expected: true,
		},
		{
			name:     "Not a PHP project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			php := &runtime.PHP{Log: logger}
			if php.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, php.Match(test.path))
			}
		})
	}
}

func TestPHPGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "PHP project",
			path:     "../testdata/php",
			expected: []any{`ARG VERSION=8.3`, regexp.MustCompile(`^ARG BUILD_CMD=$`), `ARG START_CMD="apache2-foreground`},
		},
		{
			name:     "PHP project with composer",
			path:     "../testdata/php-composer",
			expected: []any{`ARG VERSION=5.3`, `ARG INSTALL_CMD="composer update && composer install --prefer-dist --no-dev --optimize-autoloader --no-interaction"`, regexp.MustCompile(`^ARG BUILD_CMD=$`), `ARG START_CMD="apache2-foreground`},
		},
		{
			name:     "PHP project with NPM",
			path:     "../testdata/php-npm",
			expected: []any{`ARG VERSION=8.2.0`, `ARG INSTALL_CMD="yarn --frozen-lockfile"`, `ARG BUILD_CMD="yarn run build"`, `ARG START_CMD="apache2-foreground`},
		},
		{
			name:     "Not a PHP project",
			path:     "../testdata/deno",
			expected: []any{`ARG VERSION=8.3`, regexp.MustCompile(`^ARG INSTALL_CMD=$`), regexp.MustCompile(`^ARG BUILD_CMD=$`), `ARG START_CMD="apache2-foreground`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			php := &runtime.PHP{Log: logger}
			dockerfile, err := php.GenerateDockerfile(test.path)
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
