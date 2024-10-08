package runtime

import (
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Static struct {
	Log *slog.Logger
}

func (d *Static) Name() RuntimeName {
	return RuntimeNameStatic
}

func (d *Static) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "public"),
		filepath.Join(path, "static"),
		filepath.Join(path, "dist"),
		filepath.Join(path, "index.html"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Static project")
			return true
		}
	}

	d.Log.Debug("Static project not detected")
	return false
}

func (d *Static) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(staticTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	serverRoot := "."
	if _, err := os.Stat(filepath.Join(path, "index.html")); err != nil {
		roots := []string{"public", "static", "dist"}
		for _, root := range roots {
			if _, err := os.Stat(filepath.Join(path, root)); err == nil {
				serverRoot = root
				break
			}
		}
	}
	d.Log.Info("Detected root directory: " + serverRoot)

	var buf bytes.Buffer
	templateData := map[string]string{
		"ServerRoot": serverRoot,
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var staticTemplate = strings.TrimSpace(`
ARG VERSION=2
ARG BUILDER=docker.io/joseluisq/static-web-server
FROM ${BUILDER}:${VERSION}-debian
RUN apt-get update && apt-get install -y --no-install-recommends wget && apt-get clean && rm -f /var/lib/apt/lists/*_*
COPY . .

ENV PORT=8080
EXPOSE ${PORT}
ENV SERVER_PORT=${PORT}
ARG SERVER_ROOT={{.ServerRoot}}
ENV SERVER_ROOT=${SERVER_ROOT}
`)
