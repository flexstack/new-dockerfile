package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml/v2"
)

type Deno struct {
	Log *slog.Logger
}

func (d *Deno) Name() RuntimeName {
	return RuntimeNameDeno
}

func (d *Deno) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "deno.json"),
		filepath.Join(path, "deno.jsonc"),
		filepath.Join(path, "deno.lock"),
		filepath.Join(path, "deps.ts"),
		filepath.Join(path, "mod.ts"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Deno project")
			return true
		}
	}

	detected := false
	// Walk the directory to find a .ts file with a "deno.land/x" import
	filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".ts" {
			f, err := os.Open(path)
			if err != nil {
				return err
			}

			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				text := scanner.Text()

				if (strings.HasPrefix(text, "import ") || strings.HasPrefix(text, "export ")) && strings.Contains(text, " from ") && strings.Contains(text, "https://deno.land/") {
					d.Log.Info("Detected Deno project")
					detected = true
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	d.Log.Debug("Deno project not detected")
	return detected
}

func (d *Deno) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(denoTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	var denoJSON map[string]interface{}
	configFiles := []string{"deno.jsonc", "deno.json"}
	for _, file := range configFiles {
		f, err := os.Open(filepath.Join(path, file))
		if err != nil {
			continue
		}

		defer f.Close()

		if err := json.NewDecoder(f).Decode(&denoJSON); err != nil {
			return nil, fmt.Errorf("Failed to decode " + file + " file")
		}

		f.Close()
		break
	}

	var startCMD string
	var installCMD string

	scripts, ok := denoJSON["tasks"].(map[string]interface{})
	if ok {
		startCommands := []string{"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"}
		for _, cmd := range startCommands {
			if _, ok := scripts[cmd].(string); ok {
				d.Log.Info("Detected start command in deno.json: " + cmd)
				startCMD = fmt.Sprintf("deno task %s", cmd)
				break
			}
		}

		if _, ok := scripts["cache"].(string); ok {
			d.Log.Info("Detected install command in deno.json: cache")
			installCMD = "deno task cache"
		}
	}

	if startCMD == "" {
		mainFiles := []string{"mod.ts", "src/mod.ts", "main.ts", "src/main.ts", "index.ts", "src/index.ts"}
		for _, mainFile := range mainFiles {
			if _, err := os.Stat(filepath.Join(path, mainFile)); err == nil {
				d.Log.Info("Detected start command via main/mod file: " + mainFile)

				startCMD = fmt.Sprintf("deno run --allow-all %s", mainFile)
				if installCMD == "" {
					d.Log.Info("Detected install command via main/mod file: " + mainFile)
					installCMD = "deno cache " + mainFile
				}
				break
			}
		}
	}

	version, err := findDenoVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults
  Version         : %s
  Install command : %s
  Start command   : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, installCMD, startCMD),
	)

	if installCMD != "" {
		installCMDJSON, _ := json.Marshal(installCMD)
		installCMD = string(installCMDJSON)
	}

	if startCMD != "" {
		startCMDJSON, _ := json.Marshal(startCMD)
		startCMD = string(startCMDJSON)
	}

	var buf bytes.Buffer
	templateData := map[string]string{
		"Version":    *version,
		"InstallCMD": installCMD,
		"StartCMD":   startCMD,
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var denoTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDER=docker.io/denoland/deno
FROM ${BUILDER}:${VERSION} as base

FROM debian:stable-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

ENV DENO_DIR=.deno_cache
RUN mkdir -p /app/${DENO_DIR}
RUN chown -R nonroot:nonroot /app/${DENO_DIR}

COPY --chown=nonroot:nonroot --from=base /usr/bin/deno /usr/local/bin/deno
COPY --chown=nonroot:nonroot . .

USER nonroot:nonroot

ENV PORT=8080
EXPOSE ${PORT}
ARG INSTALL_CMD={{.InstallCMD}}
RUN {{.InstallMounts}}if [ ! -z "${INSTALL_CMD}" ]; then sh -c "$INSTALL_CMD"; fi

ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

func findDenoVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".mise.toml",
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
					if strings.Contains(line, "deno") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Deno version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".mise.toml":
				var mise MiseToml
				if err := toml.NewDecoder(f).Decode(&mise); err != nil {
					return nil, fmt.Errorf("Failed to decode .mise.toml file")
				}
				denoVersion, ok := mise.Tools["deno"].(string)
				if !ok {
					versions, ok := mise.Tools["deno"].([]string)
					if ok {
						denoVersion = versions[0]
					}
				}
				if denoVersion != "" {
					version = denoVersion
					log.Info("Detected Deno version in .mise.toml: " + version)
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
		version = "latest"
		log.Info(fmt.Sprintf("No Deno version detected. Using: %s", version))
	}

	return &version, nil
}
