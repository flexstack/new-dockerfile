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
)

type NextJS struct {
	Log *slog.Logger
}

func (d *NextJS) Name() RuntimeName {
	return RuntimeNameNextJS
}

func (d *NextJS) Match(path string) bool {
	checkPaths := []string{
		filepath.Join(path, "next.config.js"),
		filepath.Join(path, "next.config.ts"),
		filepath.Join(path, "next.config.cjs"),
		filepath.Join(path, "next.config.mjs"),
		filepath.Join(path, "next.config.mts"),
		filepath.Join(path, "next-env.d.ts"),
		filepath.Join(path, "src/next-env.d.ts"),
		filepath.Join(path, ".next"),
	}

	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			d.Log.Info("Detected Next.js project")
			return true
		}
	}

	d.Log.Debug("Next.js project not detected")
	return false
}

func (d *NextJS) GenerateDockerfile(path string, data ...map[string]string) ([]byte, error) {
	nextJSTemplate := nextJSServerTemplate
	nextConfigFiles := []string{
		"next.config.js",
		"next.config.ts",
		"next.config.mjs",
		"next.config.mts",
	}

	for _, file := range nextConfigFiles {
		_, err := os.Stat(filepath.Join(path, file))
		if err == nil {
			// Search for "output": "standalone" in next.config.js
			f, err := os.Open(filepath.Join(path, file))
			if err != nil {
				return nil, fmt.Errorf("Failed to open next.config.js file")
			}

			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "output") && strings.Contains(line, "standalone") {
					d.Log.Info("Found standalone output in next.config.js")
					nextJSTemplate = nextJSStandaloneTemplate
					f.Close()
					break
				}
			}

			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("Failed to read next.config.js file")
			}

			f.Close()
		}
	}

	tmpl, err := template.New("Dockerfile").Parse(nextJSTemplate)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template")
	}

	version, err := findNodeVersion(path, d.Log)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	templateData := map[string]string{
		"Version": *version,
	}
	if len(data) > 0 {
		maps.Copy(templateData, data[0])
	}
	if err := tmpl.Option("missingkey=zero").Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("Failed to execute template")
	}

	return buf.Bytes(), nil
}

var nextJSStandaloneTemplate = strings.TrimSpace(`
ARG VERSION={{.Version}}
ARG BUILDER=docker.io/library/node
FROM ${BUILDER}:${VERSION}-slim AS base

# Install dependencies only when needed
FROM base AS deps
WORKDIR /app

# Install dependencies based on the preferred package manager
COPY package.json yarn.lock* package-lock.json* pnpm-lock.yaml* bun.lockb* ./
RUN {{.InstallMounts}}if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
  elif [ -f package-lock.json ]; then npm ci; \
  elif [ -f bun.lockb ]; then npm i -g bun && bun install; \
  elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm i --frozen-lockfile; \
  else echo "Lockfile not found." && exit 1; \
  fi

# Rebuild the source code only when needed
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Next.js collects completely anonymous telemetry data about general usage.
# Learn more here: https://nextjs.org/telemetry
# Uncomment the following line in case you want to disable telemetry during the build.
ENV NEXT_TELEMETRY_DISABLED=1

RUN {{.BuildMounts}}if [ -f yarn.lock ]; then yarn run build; \
  elif [ -f package-lock.json ]; then npm run build; \
  elif [ -f bun.lockb ]; then npm i -g bun && bun run build; \
  elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm run build; \
  else echo "Lockfile not found." && exit 1; \
  fi

# Production image, copy all the files and run next
FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production
# Uncomment the following line in case you want to disable telemetry during runtime.
ENV NEXT_TELEMETRY_DISABLED 1

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

COPY --from=builder --chown=nonroot:nonroot /app/public* ./public

# Set the correct permission for prerender cache
RUN mkdir .next
RUN chown nonroot:nonroot .next

# Automatically leverage output traces to reduce image size
# https://nextjs.org/docs/advanced-features/output-file-tracing
COPY --from=builder --chown=nonroot:nonroot /app/.next/standalone ./
COPY --from=builder --chown=nonroot:nonroot /app/.next/static ./.next/static

USER nonroot

ENV PORT=3000
EXPOSE ${PORT}

# server.js is created by next build from the standalone output
# https://nextjs.org/docs/pages/api-reference/next-config-js/output
CMD HOSTNAME="0.0.0.0" node server.js
`)

var nextJSServerTemplate = strings.TrimSpace(`
ARG VERSION=lts
ARG BUILDER=docker.io/library/node
FROM ${BUILDER}:${VERSION}-slim AS base

# Install dependencies only when needed
FROM base AS deps
WORKDIR /app
COPY package.json yarn.lock* package-lock.json* pnpm-lock.yaml* bun.lockb* ./
RUN {{.InstallMounts}}if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
  elif [ -f package-lock.json ]; then npm ci; \
  elif [ -f bun.lockb ]; then npm i -g bun && bun install; \
  elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm i --frozen-lockfile; \
  else echo "Lockfile not found." && exit 1; \
  fi

FROM base AS builder

ENV NODE_ENV=production
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN {{.BuildMounts}}if [ -f yarn.lock ]; then yarn run build; \
  elif [ -f package-lock.json ]; then npm run build; \
  elif [ -f bun.lockb ]; then npm i -g bun && bun run build; \
  elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm run build; \
  else echo "Lockfile not found." && exit 1; \
  fi

# Production image, copy all the files and run next
FROM base AS runner
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && apt-get clean && rm -f /var/lib/apt/lists/*_*
RUN update-ca-certificates 2>/dev/null || true
RUN addgroup --system nonroot && adduser --system --ingroup nonroot nonroot
RUN chown -R nonroot:nonroot /app

COPY --from=builder --chown=nonroot:nonroot /app/next.config.* ./
COPY --from=builder --chown=nonroot:nonroot /app/public* ./public
COPY --from=builder --chown=nonroot:nonroot /app/.next ./.next
COPY --from=builder --chown=nonroot:nonroot /app/node_modules ./node_modules

USER nonroot

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=8080
EXPOSE ${PORT}
CMD ["node_modules/.bin/next", "start", "-H", "0.0.0.0"]
`)
