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

type Ruby struct {
	Log *slog.Logger
}

func (d *Ruby) Name() RuntimeName {
	return RuntimeNameRuby
}

func (d *Ruby) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "Gemfile"),
		filepath.Join(path, "Gemfile.lock"),
		filepath.Join(path, "Rakefile"),
		filepath.Join(path, "config.ru"),
		filepath.Join(path, "config/environment.rb"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Ruby project")
			return true
		}
	}

	d.Log.Debug("Ruby project not detected")
	return false
}

func (d *Ruby) GenerateDockerfile(path string) ([]byte, error) {
	tmpl, err := template.New("Dockerfile").Parse(rubyTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	// Parse version from go.mod
	version, err := findRubyVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	installCMD := "bundle install"
	packageManager := ""

	if _, err := os.Stat(filepath.Join(path, "package-lock.json")); err == nil {
		packageManager = "npm"
		installCMD = installCMD + " && npm ci"
	} else if _, err := os.Stat(filepath.Join(path, "pnpm-lock.yaml")); err == nil {
		packageManager = "pnpm"
		installCMD = installCMD + " && corepack enable pnpm && pnpm i --frozen-lockfile"
	} else if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
		packageManager = "yarn"
		installCMD = installCMD + " && yarn --frozen-lockfile"
	} else if _, err := os.Stat(filepath.Join(path, "bun.lockb")); err == nil {
		packageManager = "bun"
		installCMD = installCMD + " && bun install"
	}

	if packageManager != "" {
		d.Log.Info("Detected Node.js package manager: " + packageManager)
	}

	isRails := isRailsProject(path)
	buildCMD := ""
	startCMD := ""
	if isRails {
		d.Log.Info("Detected Rails project")
		buildCMD = "bundle exec rake assets:precompile"
		startCMD = "bundle exec rails server -b 0.0.0.0 -p ${PORT}"
	} else {
		configFiles := []string{"config.ru", "config/environment.rb", "Rakefile"}

		for _, fn := range configFiles {
			_, err := os.Stat(filepath.Join(path, fn))
			if err != nil {
				continue
			}

			switch fn {
			case "config.ru":
				d.Log.Info("Detected Rack project")
				startCMD = "bundle exec rackup config.ru -p ${PORT}"
			case "config/environment.rb":
				d.Log.Info("Detected Rails project")
				startCMD = "bundle exec ruby script/server"
			case "Rakefile":
				d.Log.Info("Detected Rake project")
				startCMD = "bundle exec rake"
			}

			break
		}
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  Ruby version         : %s
  Node package manager : %s
  Install command      : %s
  Build command        : %s
  Start command        : %s

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
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var rubyTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
FROM ruby:${VERSION}-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends wget && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot

ARG INSTALL_CMD={{.InstallCMD}}
ARG BUILD_CMD={{.BuildCMD}}
ENV NODE_ENV=production

RUN chown -R nonroot:nonroot /app
COPY --chown=nonroot:nonroot . .

RUN if [ ! -z "${INSTALL_CMD}" ]; then echo "${INSTALL_CMD}" > dep.sh; sh dep.sh;  fi
RUN  if [ ! -z "${BUILD_CMD}" ]; then $BUILD_CMD; fi

ENV PORT=8080
USER nonroot:nonroot

ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

func findRubyVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{
		".tool-versions",
		".ruby-version",
		"Gemfile",
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
					if strings.Contains(line, "ruby") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Ruby version in .tool-versions: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

			case ".ruby-version":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if line != "" {
						version = line
						log.Info("Detected Ruby version from .ruby-version: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read go.mod file")
				}

			case "Gemfile":
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "ruby") {
						v := strings.Split(line, "'")
						if len(v) < 2 {
							v = strings.Split(line, "\"")
						}
						ruby := v[1]
						if gteVersionRe.MatchString(ruby) {
							version = gteVersionRe.FindStringSubmatch(ruby)[1]
						} else if rangeVersionRe.MatchString(ruby) {
							version = rangeVersionRe.FindStringSubmatch(ruby)[2]
						} else if tildeVersionRe.MatchString(ruby) {
							version = tildeVersionRe.FindStringSubmatch(ruby)[1]
						} else if caretVersionRe.MatchString(ruby) {
							version = caretVersionRe.FindStringSubmatch(ruby)[1]
						} else if exactVersionRe.MatchString(ruby) {
							version = ruby
						}
						log.Info("Detected Ruby version from Gemfile: " + version)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read Gemfile")
				}

			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "3.1"
		log.Info(fmt.Sprintf("No Ruby version detected. Using: %s", version))
	}

	return &version, nil
}

func isRailsProject(path string) bool {
	_, err := os.Stat(filepath.Join(path, "Gemfile"))
	if err == nil {
		f, err := os.Open(filepath.Join(path, "Gemfile"))
		if err != nil {
			return false
		}

		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "gem 'rails'") || strings.HasPrefix(line, "gem \"rails\"") {
				return true
			}
		}
	}

	return false
}
