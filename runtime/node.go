package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Node struct {
	Log *slog.Logger
}

func (d *Node) Name() RuntimeName {
	return RuntimeNameNode
}

func (d *Node) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "yarn.lock"),
		filepath.Join(path, "package-lock.json"),
		filepath.Join(path, "pnpm-lock.yaml"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Node project")
			return true
		}
	}

	d.Log.Debug("Node project not detected")
	return false
}

func (d *Node) GenerateDockerfile(path string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(nodeTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	version, err := findNodeVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	var packageJSON map[string]interface{}

	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		f, err := os.Open(filepath.Join(path, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("Failed to open package.json file")
		}

		defer f.Close()

		if err := json.NewDecoder(f).Decode(&packageJSON); err != nil {
			return nil, fmt.Errorf("Failed to decode package.json file")
		}
	} else {
		d.Log.Info("No package.json file found")
		packageJSON = map[string]interface{}{}
	}

	installCMD := "npm ci"
	packageManager := "npm"

	if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
		installCMD = "yarn --frozen-lockfile"
		packageManager = "yarn"
	} else if _, err := os.Stat(filepath.Join(path, "pnpm-lock.yaml")); err == nil {
		installCMD = "pnpm i --frozen-lockfile"
		packageManager = "pnpm"
	}

	var buildCMD, startCMD string

	scripts, ok := packageJSON["scripts"].(map[string]interface{})

	if ok {
		d.Log.Info("Detected scripts in package.json")
		startCommands := []string{"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"}
		for _, cmd := range startCommands {
			if _, ok := scripts[cmd].(string); ok {
				startCMD = fmt.Sprintf("%s run %s", packageManager, cmd)
				d.Log.Info("Detected start command in package.json: " + startCMD)
				break
			}
		}

		if startCMD == "" {
			for name, v := range scripts {
				value, ok := v.(string)

				if ok && startScriptRe.MatchString(value) {
					startCMD = fmt.Sprintf("%s run %s", packageManager, name)
					d.Log.Info("Detected start command in package.json via regex pattern: " + startCMD)
					break
				}
			}
		}

		buildCommands := []string{"build:prod", "build:production", "build-prod", "build-production", "build"}
		for _, cmd := range buildCommands {
			if _, ok := scripts[cmd].(string); ok {
				buildCMD = fmt.Sprintf("%s run %s", packageManager, cmd)
				d.Log.Info("Detected build command in package.json: " + buildCMD)
				break
			}
		}
	}

	mainFile := ""
	if packageJSON["main"] != nil {
		mainFile = packageJSON["main"].(string)
	} else if packageJSON["module"] != nil {
		mainFile = packageJSON["module"].(string)
	}

	if startCMD == "" && mainFile != "" {
		startCMD = fmt.Sprintf("node %s", mainFile)
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  Node version    : %s
  Package manager : %s
  Install command : %s
  Build command   : %s
  Start command   : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, packageManager, installCMD, buildCMD, startCMD),
	)

	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=zero").Execute(&buf, map[string]string{
		"Version":    *version,
		"InstallCMD": safeCommand(installCMD),
		"BuildCMD":   safeCommand(buildCMD),
		"StartCMD":   safeCommand(startCMD),
	}); err != nil {
		return nil, errors.New("Failed to execute template")
	}

	return buf.Bytes(), nil
}

func safeCommand(cmd string) string {
	if cmd == "" {
		return ""
	}

	cmdJSON, _ := json.Marshal(cmd)
	return strings.ReplaceAll(string(cmdJSON), `\u0026\u0026`, "&&")
}

var startScriptRe = regexp.MustCompile(`^.*?\b(ts-)?node(mon)?\b.*?(index|main|server|client)\.([cm]?[tj]s)\b`)

var nodeTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
FROM node:${VERSION}-slim AS base
RUN corepack enable

FROM base AS deps
WORKDIR /app
COPY package.json yarn.lock* package-lock.json* pnpm-lock.yaml* bun.lockb* ./
ARG INSTALL_CMD={{.InstallCMD}}
ARG NPM_MIRROR=
RUN if [ ! -z "${NPM_MIRROR}" ]; then npm config set registry ${NPM_MIRROR}; fi
RUN if [ ! -z "${INSTALL_CMD}" ]; then echo "${INSTALL_CMD}" > dep.sh; sh dep.sh; fi

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules* ./node_modules
COPY . .
ENV NODE_ENV=production
ARG BUILD_CMD={{.BuildCMD}}
RUN  if [ ! -z "${BUILD_CMD}" ]; then sh -c "$BUILD_CMD"; fi

FROM base AS runtime
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --disabled-login --ingroup nonroot nonroot
ENV COREPACK_HOME=/app/.cache
RUN mkdir -p /app/.cache
RUN chown -R nonroot:nonroot /app

COPY --chown=nonroot:nonroot --from=builder /app .

USER nonroot:nonroot

ENV PORT=8080
ENV NODE_ENV=production
ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

func findNodeVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".nvmrc",
		".node-version",
		".tool-versions",
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
					if strings.Contains(line, "nodejs") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Node version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".nvmrc", ".node-version":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "v") {
						version = strings.TrimPrefix(line, "v")
						log.Info("Detected Node version in " + file + ": " + version)
						break
					} else {
						version = line
						log.Info("Detected Node version in " + file + ": " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read version file")
				}
			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "lts"
		log.Info(fmt.Sprintf("No Node version detected. Using %s.", version))
	}

	return &version, nil
}
