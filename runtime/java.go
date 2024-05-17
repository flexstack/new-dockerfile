package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Java struct {
	Log *slog.Logger
}

func (d *Java) Name() RuntimeName {
	return RuntimeNameJava
}

func (d *Java) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "build.gradle"),
		filepath.Join(path, "gradlew"),
		filepath.Join(path, "pom.xml"),
		filepath.Join(path, "pom.atom"),
		filepath.Join(path, "pom.clj"),
		filepath.Join(path, "pom.groovy"),
		filepath.Join(path, "pom.rb"),
		filepath.Join(path, "pom.scala"),
		filepath.Join(path, "pom.yml"),
		filepath.Join(path, "pom.yaml"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Java project")
			return true
		}
	}

	d.Log.Debug("Java project not detected")
	return false
}

func (d *Java) GenerateDockerfile(path string) ([]byte, error) {
	version, err := findJDKVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	tpl := javaMavenTemplate
	startCMD := "java $JAVA_OPTS -jar target/*jar"
	buildCMD := ""
	gradleVersion := ""

	if _, err := os.Stat(filepath.Join(path, "gradlew")); err == nil {
		gv, err := findGradleVersion(path, d.Log)
		if err != nil {
			return nil, err
		}

		gradleVersion = *gv
		tpl = javaGradleTemplate
		buildCMD = "./gradlew clean build -x check -x test"
		startCMD = "java $JAVA_OPTS -jar $(ls -1 build/libs/*jar | grep -v plain)"
	}

	mavenVersion := ""
	for _, file := range pomFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			mv, err := findMavenVersion(path, d.Log)
			if err != nil {
				return nil, err
			}

			mavenVersion = *mv
			buildCMD = "mvn -DoutputFile=target/mvn-dependency-list.log -B -DskipTests clean dependency:list install"
			break
		}
	}

	if isSpringBootApp(path) {
		d.Log.Info("Detected Spring Boot application")
		startCMD = "java -Dserver.port=${PORT} $JAVA_OPTS -jar target/*jar"
		if gradleVersion != "" {
			startCMD = "java $JAVA_OPTS -jar -Dserver.port=${PORT} $(ls -1 build/libs/*jar | grep -v plain)"
		}
	}

	if isWildflySwarmApp(path) {
		d.Log.Info("Detected Wildfly Swarm application")
		startCMD = "java -Dswarm.http.port=${PORT} $JAVA_OPTS -jar target/*jar"
	}

	d.Log.Info(
		fmt.Sprintf(`Detected defaults 
  JDK version     : %s
  Maven version   : %s
  Gradle version  : %s
  Build command   : %s
  Start command   : %s

  Docker build arguments can supersede these defaults if provided.
  See https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile`, *version, mavenVersion, gradleVersion, buildCMD, startCMD),
	)

	var buf bytes.Buffer

	tmpl, err := template.New("Dockerfile").Parse(tpl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	if buildCMD != "" {
		buildCMDJSON, _ := json.Marshal(buildCMD)
		buildCMD = string(buildCMDJSON)
	}

	if startCMD != "" {
		startCMDJSON, _ := json.Marshal(startCMD)
		startCMD = string(startCMDJSON)
	}

	if err := tmpl.Option("missingkey=zero").Execute(&buf, map[string]string{
		"Version":       *version,
		"GradleVersion": gradleVersion,
		"MavenVersion":  mavenVersion,
		"BuildCMD":      buildCMD,
		"StartCMD":      startCMD,
	}); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var javaMavenTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG MAVEN_VERSION={{.MavenVersion}}
FROM maven:${MAVEN_VERSION}-eclipse-temurin-${VERSION} AS build
WORKDIR /app

COPY pom.xml* pom.atom* pom.clj* pom.groovy* pom.rb* pom.scala* pom.yml* pom.yaml* .
RUN mvn dependency:go-offline

COPY src src
RUN mvn install

ARG BUILD_CMD={{.BuildCMD}}
RUN if [ ! -z "${BUILD_CMD}" ]; then sh -c "$BUILD_CMD"; fi

FROM eclipse-temurin:${VERSION}-jdk AS runtime
WORKDIR /app
VOLUME /tmp

RUN apt-get update && apt-get install -y --no-install-recommends wget && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

COPY --from=build --chown=nonroot:nonroot /app/target/*.jar /app/target/

ENV PORT=8080
USER nonroot:nonroot

ARG JAVA_OPTS=
ENV JAVA_OPTS=${JAVA_OPTS}
ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

var javaGradleTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG GRADLE_VERSION={{.GradleVersion}}
FROM gradle:${GRADLE_VERSION}-jdk${VERSION} AS build
WORKDIR /app

COPY build.gradle* gradlew* settings.gradle* ./
COPY gradle/ ./gradle/
COPY src src

ARG BUILD_CMD={{.BuildCMD}}
RUN if [ ! -z "${BUILD_CMD}" ]; then sh -c "$BUILD_CMD"; fi

FROM eclipse-temurin:${VERSION}-jdk AS runtime
WORKDIR /app
VOLUME /tmp

RUN apt-get update && apt-get install -y --no-install-recommends wget && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

COPY --from=build --chown=nonroot:nonroot /app/build/libs/*.jar /app/build/libs/

ENV PORT=8080
USER nonroot:nonroot

ARG JAVA_OPTS=
ENV JAVA_OPTS=${JAVA_OPTS}
ARG START_CMD={{.StartCMD}}
ENV START_CMD=${START_CMD}
RUN if [ -z "${START_CMD}" ]; then echo "Unable to detect a container start command" && exit 1; fi
CMD ${START_CMD}
`)

func findJDKVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{".tool-versions"}

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
					if strings.Contains(line, "java") {
						versionString := strings.Split(line, " ")[1]
						regexpVersion := regexp.MustCompile(`\d+`)
						version = regexpVersion.FindString(versionString)
						break
					}
				}

				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("Failed to read .tool-versions file")
				}

				log.Info("Detected JDK version in .tool-versions: " + version)
			}

			f.Close()
			if version != "" {
				break
			}
		}
	}

	if version == "" {
		version = "17"
	}

	return &version, nil
}

func findGradleVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{".tool-versions"}

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
					if strings.Contains(line, "gradle") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Gradle version in .tool-versions: " + version)
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
		version = "8"
		log.Info(fmt.Sprintf("No Gradle version detected. Using: %s", version))
	}

	return &version, nil
}

func findMavenVersion(path string, log *slog.Logger) (*string, error) {
	version := ""
	versionFiles := []string{".tool-versions"}

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
					if strings.Contains(line, "maven") {
						version = strings.Split(line, " ")[1]
						log.Info("Detected Maven version in .tool-versions: " + version)
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
		version = "3"
		log.Info(fmt.Sprintf("No Maven version detected. Using: %s", version))
	}

	return &version, nil
}

func isSpringBootApp(path string) bool {
	checkFiles := append([]string{}, pomFiles...)
	checkFiles = append(checkFiles, "build.gradle")

	for _, file := range checkFiles {
		pomXML, err := os.Open(filepath.Join(path, file))
		if err != nil {
			continue
		}

		defer pomXML.Close()
		scanner := bufio.NewScanner(pomXML)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "org.springframework.boot") {
				return true
			}
		}
	}

	return false
}

func isWildflySwarmApp(path string) bool {
	for _, file := range pomFiles {
		pomXML, err := os.Open(filepath.Join(path, file))
		if err != nil {
			continue
		}

		defer pomXML.Close()
		scanner := bufio.NewScanner(pomXML)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "wildfly-swarm") {
				return true
			} else if strings.Contains(line, "org.wildfly.swarm") {
				return true
			}
		}
	}

	return false
}

var pomFiles = []string{
	"pom.xml",
	"pom.atom",
	"pom.clj",
	"pom.groovy",
	"pom.rb",
	"pom.scala",
	"pom.yml",
	"pom.yaml",
}
