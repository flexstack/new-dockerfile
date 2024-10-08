package runtime

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml"
)

type Python struct {
	Log *slog.Logger
}

func (d *Python) Name() RuntimeName {
	return RuntimeNamePython
}

func (d *Python) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "requirements.txt"),
		filepath.Join(path, "poetry.lock"),
		filepath.Join(path, "uv.lock"),
		filepath.Join(path, "Pipfile.lock"),
		filepath.Join(path, "pyproject.toml"),
		filepath.Join(path, "pdm.lock"),
		filepath.Join(path, "main.py"),
		filepath.Join(path, "app.py"),
		filepath.Join(path, "application.py"),
		filepath.Join(path, "app/__init__.py"),
		filepath.Join(path, filepath.Base(path), "app.py"),
		filepath.Join(path, filepath.Base(path), "application.py"),
		filepath.Join(path, filepath.Base(path), "main.py"),
		filepath.Join(path, filepath.Base(path), "__init__.py"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Python project")
			return true
		}
	}

	d.Log.Debug("Python project not detected")
	return false
}

func (d *Python) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(pythonTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	// Parse version from go.mod
	version, err := findPythonVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	installCMD := ""
	packageManager := PythonPackageManagerPip
	if _, err := os.Stat(filepath.Join(path, "requirements.txt")); err == nil {
		d.Log.Info("Detected requirements.txt file")
		installCMD = "pip install --no-cache -r requirements.txt"
	} else if _, err := os.Stat(filepath.Join(path, "uv.lock")); err == nil {
		d.Log.Info("Detected a uv project")
		installCMD = "pip install uv && uv sync --python-preference=only-system --no-cache --no-dev"
		packageManager = PythonPackageManagerUv
	} else if _, err := os.Stat(filepath.Join(path, "poetry.lock")); err == nil {
		d.Log.Info("Detected a poetry project")
		installCMD = "pip install poetry && poetry install --no-dev --no-ansi --no-root"
		packageManager = PythonPackageManagerPoetry
	} else if _, err := os.Stat(filepath.Join(path, "Pipfile.lock")); err == nil {
		d.Log.Info("Detected a pipenv project")
		installCMD = "pip install pipenv && pipenv install --dev --system --deploy"
		packageManager = PythonPackageManagerPipenv
	} else if _, err := os.Stat(filepath.Join(path, "pdm.lock")); err == nil {
		d.Log.Info("Detected a pdm project")
		installCMD = "pip install pdm && pdm install --prod"
		packageManager = PythonPackageManagerPdm
	} else if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err == nil {
		d.Log.Info("Detected a pyproject.toml file")
		installCMD = "pip install --upgrade build setuptools && pip install ."
	}

	managePy := isDjangoProject(path)
	isFastAPI := isFastAPIProject(path)
	startCMD := ""
	projectName := filepath.Base(path)

	if managePy != nil {
		d.Log.Info("Detected Django project")
		startCMD = fmt.Sprintf(`python ` + *managePy + ` runserver 0.0.0.0:${PORT}`)
	} else if !isFastAPI {
		if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err == nil {
			f, err := os.Open(filepath.Join(path, "pyproject.toml"))
			if err == nil {
				var pyprojectTOML map[string]interface{}
				err := toml.NewDecoder(f).Decode(&pyprojectTOML)
				if err == nil {
					if project, ok := pyprojectTOML["project"].(map[string]interface{}); ok {
						if name, ok := project["name"].(string); ok {
							projectName = name
						}
					} else if project, ok := pyprojectTOML["tool.poetry"].(map[string]interface{}); ok {
						if name, ok := project["name"].(string); ok {
							projectName = name
						}
					}
				}

				if projectName != "" {
					startCMD = fmt.Sprintf(`python -m %s`, projectName)
					d.Log.Info("Detected start command via pyproject.toml")
				}
			}
		}
	}

	if startCMD == "" {
		mainFiles := []string{
			"main.py",
			"app.py",
			"application.py",
			"app/main.py",
			"app/__init__.py",
			filepath.Join(path, filepath.Base(path), "main.py"),
			filepath.Join(path, filepath.Base(path), "app.py"),
			filepath.Join(path, filepath.Base(path), "application.py"),
			filepath.Join(path, filepath.Base(path), "__init__.py"),
		}

		for _, fn := range mainFiles {
			_, err := os.Stat(filepath.Join(path, fn))
			if err != nil {
				continue
			}

			if isFastAPI {
				startCMD = fmt.Sprintf(`fastapi run %s --port ${PORT}`, fn)
			} else {
				startCMD = fmt.Sprintf(`python %s`, fn)
				d.Log.Info("Detected start command via main file: " + startCMD)
			}
			break
		}
	}

	packagerInstructions := ""
	switch packageManager {
	case PythonPackageManagerPoetry:
		packagerInstructions = poetryInstructions
	case PythonPackageManagerUv:
		packagerInstructions = uvInstructions
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  Python version       : %s
  Install command      : %s
  Start command        : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, installCMD, startCMD),
	)

	var buf bytes.Buffer
	templateData := map[string]string{
		"Version":              *version,
		"InstallCMD":           safeCommand(installCMD),
		"StartCMD":             safeCommand(startCMD),
		"PackagerInstructions": packagerInstructions,
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var pythonTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDER=docker.io/library/python
FROM ${BUILDER}:${VERSION}-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app
RUN mkdir -p /var/cache
RUN chown -R nonroot:nonroot /var/cache

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
{{ .PackagerInstructions }}

COPY --chown=nonroot:nonroot . .
ARG INSTALL_CMD={{.InstallCMD}}
RUN if [ ! -z "${INSTALL_CMD}" ]; then sh -c "$INSTALL_CMD";  fi

ENV PORT=8080
EXPOSE ${PORT}
USER nonroot:nonroot

ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

var poetryInstructions = `
ENV POETRY_NO_INTERACTION=1
ENV POETRY_VIRTUALENVS_CREATE=false
ENV POETRY_CACHE_DIR='/var/cache/pypoetry'
ENV POETRY_HOME='/usr/local'`

var uvInstructions = `
# Set the UV_CACHE_DIR environment variable to a directory where uv will store its cache
ENV UV_CACHE_DIR='/var/cache/uv'
# Use the virtual environment automatically
ENV VIRTUAL_ENV=/app/.venv
ENV PATH="/app/.venv/bin:$PATH"`

func findPythonVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".python-version",
		".mise.toml",
		"runtime.txt",
	}

	for _, file := range versionFiles {
		fp := filepath.Join(path, file)
		_, err := os.Stat(fp)

		if err == nil {
			f, err := os.Open(fp)
			if err != nil {
				continue
			}

			defer f.Close()
			switch file {
			case ".tool-versions":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "python") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Python version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".python-version":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if line != "" {
						version = line
						log.Info("Detected Python version from .python-version: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .python-version file")
				}

			case "runtime.txt":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "python-") {
						version = strings.TrimPrefix(line, "python-")
						log.Info("Detected Python version from runtime.txt: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read runtime.txt file")
				}

			case ".mise.toml":
				var mise MiseToml
				if err := toml.NewDecoder(f).Decode(&mise); err != nil {
					return nil, fmt.Errorf("Failed to decode .mise.toml file")
				}
				pythonVersion, ok := mise.Tools["python"].(string)
				if !ok {
					versions, ok := mise.Tools["python"].([]string)
					if ok {
						pythonVersion = versions[0]
					}
				}
				if pythonVersion != "" {
					version = pythonVersion
					log.Info("Detected Python version in .mise.toml: " + version)
					break
				}
			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "3.12"
		log.Info(fmt.Sprintf("No Python version detected. Using %s.", version))
	}

	return &version, nil
}

func isDjangoProject(path string) *string {
	manageFiles := []string{"manage.py", "app/manage.py", filepath.Join(filepath.Base(path), "manage.py")}
	var managePy *string
	for _, file := range manageFiles {
		_, err := os.Stat(filepath.Join(path, file))
		if err == nil {
			managePy = &file
			break
		}
	}

	if managePy == nil {
		return nil
	}

	packagerFiles := []string{"requirements.txt", "pyproject.toml", "Pipfile"}

	for _, file := range packagerFiles {
		_, err := os.Stat(filepath.Join(path, file))
		if err == nil {
			f, err := os.Open(filepath.Join(path, file))
			if err != nil {
				return nil
			}

			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(strings.ToLower(line), "django") {
					return managePy
				}
			}

			f.Close()
		}
	}

	return nil
}

func isFastAPIProject(path string) bool {
	packagerFiles := []string{"requirements.txt", "pyproject.toml", "Pipfile"}

	for _, file := range packagerFiles {
		_, err := os.Stat(filepath.Join(path, file))
		if err == nil {
			f, err := os.Open(filepath.Join(path, file))
			if err != nil {
				return false
			}

			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(strings.ToLower(line), "fastapi") {
					return true
				}
			}

			f.Close()
		}
	}

	return false
}

type PythonPackageManager string

const (
	PythonPackageManagerPip    PythonPackageManager = "pip"
	PythonPackageManagerPoetry PythonPackageManager = "poetry"
	PythonPackageManagerUv     PythonPackageManager = "uv"
	PythonPackageManagerPipenv PythonPackageManager = "pipenv"
	PythonPackageManagerPdm    PythonPackageManager = "pdm"
)
