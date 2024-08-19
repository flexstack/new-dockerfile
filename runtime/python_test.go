package runtime_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/flexstack/new-dockerfile/runtime"
)

func TestPythonMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Python project",
			path:     "../testdata/python",
			expected: true,
		},
		{
			name:     "Python project with django",
			path:     "../testdata/python-django",
			expected: true,
		},
		{
			name:     "Python project with pdm",
			path:     "../testdata/python-pdm",
			expected: true,
		},
		{
			name:     "Python project with poetry",
			path:     "../testdata/python-poetry",
			expected: true,
		},
		{
			name:     "Python project with pyproject",
			path:     "../testdata/python-pyproject",
			expected: true,
		},
		{
			name:     "Not a Python project",
			path:     "../testdata/deno",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			python := &runtime.Python{Log: logger}
			if python.Match(test.path) != test.expected {
				t.Errorf("expected %v, got %v", test.expected, python.Match(test.path))
			}
		})
	}
}

func TestPythonGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name: "Python project",
			path: "../testdata/python",
			expected: []any{
				`ARG VERSION=3.12`,
				`ARG INSTALL_CMD="pip install -r requirements.txt"`,
				`ARG START_CMD="python main.py"`,
			},
		},
		{
			name: "Python project with django",
			path: "../testdata/python-django",
			expected: []any{
				`ARG VERSION=3.6.0`,
				`ARG INSTALL_CMD="pip install pipenv && pipenv install --dev --system --deploy"`,
				`ARG START_CMD="python manage.py runserver 0.0.0.0:${PORT}"`,
			},
		},
		{
			name: "Python project with pdm",
			path: "../testdata/python-pdm",
			expected: []any{
				`ARG VERSION=3.4.1`,
				`ARG INSTALL_CMD="pip install pdm && pdm install --prod"`,
				`ARG START_CMD="python app.py"`,
			},
		},
		{
			name: "Python project with poetry",
			path: "../testdata/python-poetry",
			expected: []any{
				`ARG VERSION=3.8.5`,
				`ARG INSTALL_CMD="pip install poetry && poetry install --no-dev --no-ansi --no-root"`,
				`ARG START_CMD="python app/main.py"`,
			},
		},
		{
			name: "Python project with pyproject",
			path: "../testdata/python-pyproject",
			expected: []any{
				`ARG VERSION=3.12`,
				`ARG INSTALL_CMD="pip install --upgrade build setuptools && pip install .`,
				`ARG START_CMD="python -m pyproject"`,
			},
		},
		{
			name: "Python project with FastAPI",
			path: "../testdata/python-fastapi",
			expected: []any{
				`ARG VERSION=3.6.0`,
				`ARG INSTALL_CMD="pip install pipenv && pipenv install --dev --system --deploy"`,
				`ARG START_CMD="fastapi run main.py --port ${PORT}"`,
			},
		},
		{
			name: "Not a Python project",
			path: "../testdata/deno",
			expected: []any{
				`ARG VERSION=3.12`,
				regexp.MustCompile(`^ARG INSTALL_CMD=$`),
				regexp.MustCompile(`^ARG START_CMD=$`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			python := &runtime.Python{Log: logger}
			dockerfile, err := python.GenerateDockerfile(test.path)
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
