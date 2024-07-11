package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Bun struct {
	Log *slog.Logger
}

func (d *Bun) Name() RuntimeName {
	return RuntimeNameBun
}

func (d *Bun) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "bun.lockb"),
		filepath.Join(path, "bunfig.toml"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Bun project")
			return true
		}
	}

	d.Log.Debug("Bun project not detected")
	return false
}

func (d *Bun) GenerateDockerfile(path string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(bunTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	var packageJSON map[string]interface{}
	configFiles := []string{"package.json"}
	for _, file := range configFiles {
		f, err := os.Open(filepath.Join(path, file))
		if err != nil {
			continue
		}

		defer f.Close()

		if err := json.NewDecoder(f).Decode(&packageJSON); err != nil {
			return nil, fmt.Errorf("Failed to decode " + file + " file")
		}

		f.Close()
		break
	}

	var startCMD, buildCMD string

	scripts, ok := packageJSON["scripts"].(map[string]interface{})
	if ok {
		d.Log.Info("Detected scripts in package.json")

		startCommands := []string{"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"}
		for _, cmd := range startCommands {
			if _, ok := scripts[cmd].(string); ok {
				d.Log.Info("Detected start command in package.json: " + cmd)
				startCMD = fmt.Sprintf("bun run %s", cmd)
				break
			}
		}

		buildCommands := []string{"build:prod", "build:production", "build-prod", "build-production", "build"}
		for _, cmd := range buildCommands {
			if _, ok := scripts[cmd].(string); ok {
				d.Log.Info("Detected build command in package.json: " + cmd)
				buildCMD = fmt.Sprintf("bun run %s", cmd)
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
		d.Log.Info("Detected start command via main file: " + mainFile)
		startCMD = fmt.Sprintf("bun %s", mainFile)
	}

	version, err := findBunVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults
  Version         : %s
  Install command : bun install
  Build command   : %s
  Start command   : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, buildCMD, startCMD),
	)

	if startCMD != "" {
		startCMDJSON, _ := json.Marshal(startCMD)
		startCMD = string(startCMDJSON)
	}

	if buildCMD != "" {
		buildCMDJSON, _ := json.Marshal(buildCMD)
		buildCMD = string(buildCMDJSON)
	}

	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=zero").Execute(&buf, map[string]string{
		"Version":  *version,
		"BuildCMD": buildCMD,
		"StartCMD": startCMD,
	}); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var bunTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDER=docker.io/oven/bun
FROM ${BUILDER}:${VERSION} AS base

FROM base AS deps
WORKDIR /app
COPY package.json bun.lockb ./
ARG INSTALL_CMD="bun install"
RUN if [ ! -z "${INSTALL_CMD}" ]; then sh -c "$INSTALL_CMD"; fi

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules* ./node_modules
COPY . .
ENV NODE_ENV=production
ARG BUILD_CMD={{.BuildCMD}}
RUN  if [ ! -z "${BUILD_CMD}" ]; then sh -c "$BUILD_CMD"; fi

FROM ${BUILDER}:${VERSION}-slim AS runtime
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
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

func findBunVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
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
					if strings.Contains(line, "bun") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Bun version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "1"
		log.Info(fmt.Sprintf("No Bun version detected. Using: %s", version))
	}

	return &version, nil
}
