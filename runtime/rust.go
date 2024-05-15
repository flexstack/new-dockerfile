package runtime

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml"
)

type Rust struct {
	Log *slog.Logger
}

func (d *Rust) Name() RuntimeName {
	return RuntimeNameRust
}

func (d *Rust) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "Cargo.toml"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Rust project")
			return true
		}
	}

	d.Log.Debug("rust project not detected")
	return false
}

func (d *Rust) GenerateDockerfile(path string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(rustlangTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	var binName string
	// Parse the Cargo.toml file to get the binary name
	cargoTomlPath := filepath.Join(path, "Cargo.toml")
	if _, err := os.Stat(cargoTomlPath); err == nil {
		f, err := os.Open(cargoTomlPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to open Cargo.toml")
		}

		defer f.Close()

		var cargoTOML map[string]interface{}
		if err := toml.NewDecoder(f).Decode(&cargoTOML); err != nil {
			return nil, fmt.Errorf("Failed to decode Cargo.toml")
		}

		checkBins := []string{"bin", "lib", "package"}
		var ok bool
		var pkg map[string]interface{}
		for _, bin := range checkBins {
			// [[bin]]
			// [lib]
			// [package]
			if bin == "bin" {
				if pkgs, ok := cargoTOML[bin].([]map[string]interface{}); ok {
					if len(pkgs) > 0 {
						d.Log.Info("Detected binary in Cargo.toml via [[bin]]")
						pkg = pkgs[0]
						break
					}
				}
			} else if pkg, ok = cargoTOML[bin].(map[string]interface{}); ok {
				d.Log.Info("Detected binary in Cargo.toml via [" + bin + "]")
				break
			}
		}

		if binName, ok = pkg["name"].(string); !ok {
			d.Log.Warn("Failed to get binary name from Cargo.toml")
		} else {
			d.Log.Info("Detected binary name: " + binName)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=zero").Execute(&buf, map[string]string{
		"BinName": binName,
	}); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var rustlangTemplate = strings.TrimSpace(`
FROM --platform=${BUILDPLATFORM} messense/cargo-zigbuild:latest AS build
WORKDIR /app
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN if [ "${TARGETARCH}" = "amd64" ]; then rustup target add x86_64-unknown-linux-gnu; else rustup target add aarch64-unknown-linux-gnu; fi
RUN if [ "${TARGETARCH}" = "amd64" ]; then cargo zigbuild --release --target x86_64-unknown-linux-gnu; else cargo zigbuild --release --target aarch64-unknown-linux-gnu; fi

FROM debian:stable-slim AS runtime
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends wget && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

ARG BIN_NAME={{.BinName}}
ENV BIN_NAME=${BIN_NAME}
COPY --chown=nonroot:nonroot --from=build /app/target/*/release/${BIN_NAME} ./app

USER nonroot:nonroot

ENV PORT=8080
CMD ["/app/app"]
`)
