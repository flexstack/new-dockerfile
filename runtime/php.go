package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type PHP struct {
	Log *slog.Logger
}

func (d *PHP) Name() RuntimeName {
	return RuntimeNamePHP
}

func (d *PHP) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "composer.json"),
		filepath.Join(path, "index.php"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected PHP project")
			return true
		}
	}

	d.Log.Debug("PHP project not detected")
	return false
}

func (d *PHP) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(phpTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	// Parse version from go.mod
	version, err := findPHPVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	startCMD := "apache2-foreground"
	installCMD := ""
	if _, err := os.Stat(filepath.Join(path, "composer.json")); err == nil {
		d.Log.Info("Detected composer.json file")
		installCMD = "composer update && composer install --prefer-dist --no-dev --optimize-autoloader --no-interaction"
	}

	packageManager := ""
	npmInstallCMD := ""
	if _, err := os.Stat(filepath.Join(path, "package-lock.json")); err == nil {
		packageManager = "npm"
		npmInstallCMD = "npm ci"
	} else if _, err := os.Stat(filepath.Join(path, "pnpm-lock.yaml")); err == nil {
		packageManager = "pnpm"
		npmInstallCMD = "corepack enable pnpm && pnpm i --frozen-lockfile"
	} else if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
		packageManager = "yarn"
		npmInstallCMD = "yarn --frozen-lockfile"
	} else if _, err := os.Stat(filepath.Join(path, "bun.lockb")); err == nil {
		packageManager = "bun"
		npmInstallCMD = "bun install"
	}

	if npmInstallCMD != "" {
		d.Log.Info("Detected package-lock.json, pnpm-lock.yaml, yarn.lock or bun.lockb file")
		if installCMD == "" {
			installCMD = npmInstallCMD
		} else {
			installCMD = fmt.Sprintf("%s && %s", installCMD, npmInstallCMD)
		}
	}

	buildCMD := ""
	if packageManager != "" {
		f, err := os.Open(filepath.Join(path, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("Failed to open package.json file")
		}

		defer f.Close()

		var packageJSON map[string]interface{}
		if err := json.NewDecoder(f).Decode(&packageJSON); err != nil {
			return nil, fmt.Errorf("Failed to decode package.json file")
		}

		scripts, ok := packageJSON["scripts"].(map[string]interface{})
		if ok {
			d.Log.Info("Detected scripts in package.json")
			buildCommands := []string{"build:prod", "build:production", "build-prod", "build-production", "build"}
			for _, cmd := range buildCommands {
				if _, ok := scripts[cmd].(string); ok {
					corepack := ""
					if packageManager == "pnpm" {
						corepack = "corepack enable pnpm &&"
					}
					buildCMD = fmt.Sprintf("%s%s run %s", corepack, packageManager, cmd)
					d.Log.Info("Detected build command in package.json: " + buildCMD)
					break
				}
			}
		}

		f.Close()
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  PHP version     : %s
  Install command : %s
  Build command   : %s
  Start command   : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, installCMD, buildCMD, startCMD),
	)

	var buf bytes.Buffer
	templateData := map[string]string{
		"Version":    *version,
		"InstallCMD": safeCommand(installCMD),
		"BuildCMD":   safeCommand(buildCMD),
		"StartCMD":   safeCommand(startCMD),
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var phpTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDER=docker.io/library/composer
FROM ${BUILDER}:lts as build
RUN apk add --no-cache nodejs npm
WORKDIR /app
COPY . .

ARG INSTALL_CMD={{.InstallCMD}}
ARG BUILD_CMD={{.BuildCMD}}
RUN {{.InstallMounts}}if [ ! -z "${INSTALL_CMD}" ]; then sh -c "$INSTALL_CMD"; fi
RUN {{.BuildMounts}}if [ ! -z "${BUILD_CMD}" ]; then sh -c "$BUILD_CMD"; fi

FROM php:${VERSION}-apache AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot	
	
ENV PORT=8080
EXPOSE ${PORT}
RUN sed -i "s/80/${PORT}/g" /etc/apache2/sites-available/000-default.conf /etc/apache2/ports.conf
COPY --from=build --chown=nonroot:nonroot /app /var/www/html

USER nonroot:nonroot

ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

func findPHPVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		"composer.json",
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
					if strings.Contains(line, "php") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected PHP version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case "composer.json":
				var composerJSON map[string]interface{}
				err := json.NewDecoder(f).Decode(&composerJSON)
				if err != nil {
					return nil, fmt.Errorf("Failed to read composer.json file")
				}

				if require, ok := composerJSON["require"].(map[string]interface{}); ok {
					if php, ok := require["php"].(string); ok {
						// Version can be a range, e.g. ">=7.2" so we need to extract the version
						if gteVersionRe.MatchString(php) {
							version = gteVersionRe.FindStringSubmatch(php)[1]
						} else if rangeVersionRe.MatchString(php) {
							version = rangeVersionRe.FindStringSubmatch(php)[2]
						} else if tildeVersionRe.MatchString(php) {
							version = tildeVersionRe.FindStringSubmatch(php)[1]
						} else if caretVersionRe.MatchString(php) {
							version = caretVersionRe.FindStringSubmatch(php)[1]
						} else if exactVersionRe.MatchString(php) {
							version = exactVersionRe.FindStringSubmatch(php)[1]
						}

						version = strings.TrimSuffix(version, ".")
						log.Info("Detected PHP version from composer.json: " + version)
					}
				}
			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "8.3"
		log.Info(fmt.Sprintf("No PHP version detected. Using: %s", version))
	}

	return &version, nil
}

var gteVersionRe = regexp.MustCompile(`^>=\s*([\d.]+)`)
var rangeVersionRe = regexp.MustCompile(`^([\d.]+)\s*-\s*([\d.]+)`)
var tildeVersionRe = regexp.MustCompile(`^~\s*([\d.]+)`)
var caretVersionRe = regexp.MustCompile(`^\^([\d.]+)`)
var exactVersionRe = regexp.MustCompile(`^([\d.]+)`)
