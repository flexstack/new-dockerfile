package runtime

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Elixir struct {
	Log *slog.Logger
}

func (d *Elixir) Name() RuntimeName {
	return RuntimeNameElixir
}

func (d *Elixir) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "mix.exs"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Elixir project")
			return true
		}
	}

	d.Log.Debug("Elixir project not detected")
	return false
}

func (d *Elixir) GenerateDockerfile(path string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(elixirTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	// Parse elixirVersion from go.mod
	elixirVersion, err := findElixirVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	otpVersion, err := findOTPVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	binName, err := findBinName(path)
	if err != nil {
		return nil, err
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  Elixir version : %s
  Erlang version : %s
  Binary name    : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *elixirVersion, *otpVersion, binName),
	)

	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=zero").Execute(&buf, map[string]string{
		"ElixirVersion": *elixirVersion,
		"OTPVersion":    strings.Split(*otpVersion, ".")[0],
		"BinName":       binName,
	}); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var elixirTemplate = strings.TrimSpace(`
ARG VERSION={{.ElixirVersion}}
ARG OTP_VERSION={{.OTPVersion}}
FROM elixir:${VERSION}-otp-${OTP_VERSION}-slim AS build
WORKDIR /app
RUN apt-get update -y && apt-get install -y build-essential git \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

ENV MIX_ENV=prod
RUN mix local.hex --force && mix local.rebar --force

COPY mix.exs mix.lock ./
RUN mix deps.get --only $MIX_ENV
RUN mkdir config

COPY config/config.exs config/${MIX_ENV}.exs config/
RUN mix deps.compile

COPY priv priv
COPY lib lib
COPY assets assets
RUN mix assets.deploy 
RUN mix compile

COPY config/runtime.exs config/
RUN mix release

FROM debian:stable-slim AS runtime
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends wget libstdc++6 openssl libncurses5 locales ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot

RUN chown -R nonroot:nonroot /app
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

ENV MIX_ENV="prod"

# Only copy the final release from the build stage
ARG BIN_NAME={{.BinName}}
ENV BIN_NAME=${BIN_NAME}
RUN if [ -z "${BIN_NAME}" ]; then echo "Unable to detect an app name" && exit 1; fi
COPY --from=build --chown=nonroot:nonroot /app/_build/${MIX_ENV}/rel/${BIN_NAME} ./
RUN cp /app/bin/${BIN_NAME} /app/bin/server

ENV PORT=8080
USER nonroot:nonroot

CMD ["/app/bin/server", "start"]
`)

func findElixirVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".elixir-version",
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
					if strings.Contains(line, "elixir") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Elixir version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".elixir-version":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if line != "" {
						version = line
						log.Info("Detected Elixir version from .elixir-version: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .elixir-version file")
				}
			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "1.12"
		log.Info(fmt.Sprintf("No Elixir version detected. Using: %s", version))
	}

	return &version, nil
}

func findOTPVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".erlang-version",
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
					if strings.Contains(line, "erlang") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Erlang version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".erlang-version":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if line != "" {
						version = line
						log.Info("Detected Erlang version from .erlang-version: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .erlang-version file")
				}

			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "26.2.5"
		log.Info(fmt.Sprintf("No Erlang version detected. Using: %s", version))
	}

	return &version, nil
}

func isPhoenixProject(path string) bool {
	_, err := os.Stat(filepath.Join(path, "config/config.exs"))
	return err == nil
}

func findBinName(path string) (string, error) {
	if _, err := os.Stat(filepath.Join(path, "mix.exs")); err != nil {
		return "", nil
	}

	configFile, err := os.Open(filepath.Join(path, "mix.exs"))
	if err != nil {
		return "", err
	}

	defer configFile.Close()

	scanner := bufio.NewScanner(configFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "app: :") {
			binName := strings.Split(strings.Replace(line, "app:", "", 1), ":")[1]
			binName = strings.TrimSpace(strings.Trim(binName, ",'\""))
			return binName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
