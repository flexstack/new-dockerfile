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

	"github.com/pelletier/go-toml/v2"
)

type Golang struct {
	Log *slog.Logger
}

func (d *Golang) Name() RuntimeName {
	return RuntimeNameGolang
}

func (d *Golang) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "go.mod"),
		filepath.Join(path, "main.go"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Golang project")
			return true
		}
	}

	d.Log.Debug("Golang project not detected")
	return false
}

func (d *Golang) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(golangTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	// Parse version from go.mod
	version, err := findGoVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	pkg := ""
	stat, err := os.Stat(filepath.Join(path, "cmd"))
	if err == nil {
		if stat.IsDir() {
			d.Log.Info("Found cmd directory. Detecting package...")

			// Walk the directory to find the main package
			items, err := os.ReadDir(filepath.Join(path, "cmd"))
			if err != nil {
				return nil, fmt.Errorf("Failed to read cmd directory")
			}

			for _, item := range items {
				if !item.IsDir() {
					if item.Name() == "main.go" {
						pkg = "./" + filepath.Join("cmd", item.Name())
						break
					}

					continue
				}

				pkg = "./" + filepath.Join("cmd", item.Name())
				break
			}
		}
	}

	if pkg == "" {
		if _, err := os.Stat(filepath.Join(path, "main.go")); err == nil {
			pkg = "./main.go"
		}
	}

	d.Log.Info("Using package: " + pkg)
	var buf bytes.Buffer
	templateData := map[string]string{
		"Version": *version,
		"Package": pkg,
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var golangTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDPLATFORM=linux/amd64
ARG BUILDER=docker.io/library/golang
FROM --platform=${BUILDPLATFORM} ${BUILDER}:${VERSION} AS base

FROM base AS deps 
WORKDIR /go/src/app
COPY go.mod* go.sum* ./
# GOPROXY is used to specify the module proxy to use.
ARG GOPROXY=direct
ENV GOPROXY=${GOPROXY}
RUN if [ -f go.mod ]; then go mod download && go mod tidy; fi

FROM deps AS build
WORKDIR /go/src/app

COPY . .

ARG PACKAGE={{.Package}}
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0
# -trimpath removes the absolute path to the source code in the binary
# -ldflags="-s -w" removes the symbol table and debug information from the binary
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /go/bin/app "${PACKAGE}"

FROM debian:stable-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

COPY --chown=nonroot:nonroot --from=build /go/bin/app .

ENV PORT=8080
EXPOSE ${PORT}
USER nonroot:nonroot
CMD ["/app/app"]
`)

func findGoVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".mise.toml",
		"go.mod",
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
					if strings.Contains(line, "golang") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Go version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case "go.mod":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "go ") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Go version in go.mod: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read go.mod file")
				}

			case ".mise.toml":
				var mise MiseToml
				if err := toml.NewDecoder(f).Decode(&mise); err != nil {
					return nil, fmt.Errorf("Failed to decode .mise.toml file")
				}
				goVersion, ok := mise.Tools["go"].(string)
				if !ok {
					versions, ok := mise.Tools["go"].([]string)
					if ok {
						goVersion = versions[0]
					}
				}
				if goVersion != "" {
					version = goVersion
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
		version = "1.17"
		log.Info(fmt.Sprintf("No Go version detected. Using: %s", version))
	}

	return &version, nil
}
