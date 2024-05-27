# Dockerfile Generator

`new-dockerfile` is a CLI tool and Go package automatically generates a configurable Dockerfile 
based on your project source code. It supports a wide range of languages and frameworks, including Next.js, 
Node.js, Python, Ruby, Java/Spring Boot, Go, Elixir/Phoenix, and more.

See the [FlexStack Documentation](https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile) page for FlexStack-specific documentation
related to this tool.

## Features

- [x] Automatically detect the runtime and framework used by your project
- [x] Use version managers like [asdf](https://github.com/asdf-vm), nvm, rbenv, and pyenv to install the correct version of the runtime
- [x] Make a best effort to detect any install, build, and start commands
- [x] Generate a Dockerfile with sensible defaults that are configurable via [Docker Build Args](https://docs.docker.com/build/guide/build-args/)
- [x] Support for a wide range of the most popular languages and frameworks including Next.js, Phoenix, Spring Boot, Django, and more
- [x] Use Debian Slim as the runtime image for a smaller image size and better security, while still supporting the most common dependencies and avoiding deployment headaches caused by Alpine Linux gotchas
- [x] Includes `wget` in the runtime image for adding health checks to services, e.g. `wget -nv -t1 --spider 'http://localhost:8080/healthz' || exit 1`
- [x] Includes `ca-certificates` in the runtime image to allow secure HTTPS connections
- [x] Use multi-stage builds to reduce the size of the final image
- [x] Run the application as a non-root user for better security
- [x] Supports multi-platform images that run on both x86 and ARM CPU architectures

## Supported Runtimes

- [Bun](#bun)
- [Deno](#deno)
- [Elixir](#elixir)
- [Go](#go)
- [Java](#java)
- [Next.js](#nextjs)
- [Node.js](#nodejs)
- [PHP](#php)
- [Python](#python)
- [Ruby](#ruby)
- [Rust](#rust)
- [Static](#static-html-css-js) (HTML, CSS, JS)

### Additional runtimes we'd like to support

- C#/.NET
- C++
- Scala
- Zig

[Consider contributing](CONTRIBUTING.md) to add support for these or any other runtimes!

## Installation

### cURL

```sh
curl -fsSL https://flexstack.com/install/new-dockerfile | bash
```

### Go Package

```sh
go get github.com/flexstack/new-dockerfile
```

## CLI Usage

```sh
new-dockerfile [options]
```

## CLI Options

- `--path` - Path to the project source code (default: `.`)
- `--write` - Write the generated Dockerfile to the project at the specified path (default: `false`)
- `--runtime` - Force a specific runtime, e.g. `node` (default: `auto`)
- `--quiet` - Disable all logging except for errors (default: `false`)
- `--help` - Show help

## CLI Examples

Print the generated Dockerfile to the console:
```sh
new-dockerfile
```

Write a Dockerfile to the current directory:
```sh
new-dockerfile --write
```

Write a Dockerfile to a specific directory:
```sh
new-dockerfile > path/to/Dockerfile
```

Force a specific runtime:
```sh
new-dockerfile --runtime next.js
```

List the supported runtimes:
```sh
new-dockerfile --runtime list
```

## Read from Config file

In the CI use case, you might need a very common step for generating a `Dockerfile`. You can create a config file for the
CLI options. The default config file name is `new-dockerfile.yaml`, and it should be in the root directory of your git
repository. Especially, there are multiple kinds of files, `new-dockerfile` might not be able to it a correct one.

```yaml
runtime: go
```

And, the CLI option will overwrite the values from config file.

## How it Works

The tool searches for common files and directories in your project to determine the runtime and framework.
For example, if it finds a `package.json` file, it will assume the project is a Node.js project unless
a `next.config.js` file is present, in which case it will assume the project is a Next.js project.

From there, it will read any `.tool-versions` or other version manager files to determine the correct version
of the runtime to install. It will then make a best effort to detect any install, build, and start commands.
For example, a `serve`, `start`, `start:prod` command in a `package.json` file will be used as the start command.

Runtimes are matched against in the order they appear when you run `new-dockerfile --runtime list`.

Read on to see runtime-specific examples and how to configure the generated Dockerfile.

## Runtime Documentation

### Bun

[Bun](https://bun.sh/) is a fast JavaScript all-in-one toolkit

#### Detected Files
  - `bun.lockb`
  - `bunfig.toml`

#### Version Detection
  - `.tool-versions` - `bun {VERSION}`

#### Runtime Image
`oven/bun:${VERSION}-slim`

#### Build Args
  - `VERSION` - The version of Bun to install (default: `1`)
  - `INSTALL_CMD` - The command to install dependencies (default: `bun install`)
  - `BUILD_CMD` - The command to build the project (default: detected from `package.json`)
  - `START_CMD` - The command to start the project (default: detected from `package.json`)

#### Build Command

Detected in order of precedence:
  - `package.json` scripts: `"build:prod", "build:production", "build-prod", "build-production", "build"`

#### Start Command

Detected in order of precedence:
  - `package.json` scripts: `"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"`
  - `package.json` main/module file: `bun run ${mainFile}`

---

### Deno

[Deno](https://deno.com/) is a secure runtime for JavaScript with native TypeScript and JSX support

#### Detected Files
  - `deno.jsonc`
  - `deno.json`
  - `deno.lock`
  - `deps.ts`
  - `mod.ts`

#### Version Detection
  - `.tool-versions` - `deno {VERSION}`

#### Runtime Image
`debian:stable-slim`

#### Build Args
  - `VERSION` - The version of Deno to install (default: `latest`)
  - `INSTALL_CMD` - The command to install dependencies (default: detected from `deno.jsonc` and source code)
  - `START_CMD` - The command to start the project (default: detected from `deno.jsonc` and source code)

#### Install Command

Detected in order of precedence:
  - `deno.jsonc` tasks: `"cache"`
  - Main/module file: `deno cache ["mod.ts", "src/mod.ts", "main.ts", "src/main.ts", "index.ts", "src/index.ts]"`

#### Start Command

Detected in order of precedence:
  - `deno.jsonc` tasks: `"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"`
  - Main/module file: `deno run ["mod.ts", "src/mod.ts", "main.ts", "src/main.ts", "index.ts", "src/index.ts]"`
  
---

### Elixir

[Elixir](https://elixir-lang.org/) is a dynamic, functional language designed for building scalable and maintainable applications.

#### Detected Files
  - `mix.exs`

#### Version Detection
  - `.tool-versions` - `elixir {VERSION}`
  - `.tool-versions` - `erlang {VERSION}`
  - `.elixir-version` - `{VERSION}`
  - `.erlang-version` - `{VERSION}`

#### Runtime Image
`debian:stable-slim`

#### Build Args
  - `VERSION` - The version of Elixir to install (default: `1.12`)
  - `OTP_VERSION` - The version of Erlang to install (default: `26.2.5`)
  - `BIN_NAME` - The name of the release binary (default: detected via app name in `mix.exs`)

#### Start Command
`/app/bin/{BIN_NAME} start`

---

### Go

[Go](https://golang.org/) is an open-source programming language that makes it easy to build simple, reliable, and efficient software.

#### Detected Files
  - `go.mod`
  - `main.go`

#### Version Detection
  - `.tool-versions` - `golang {VERSION}`
  - `go.mod` - `go {VERSION}`

#### Runtime Image
`debian:stable-slim`

#### Build Args
  - `VERSION` - The version of Go to install (default: `1.17`)
  - `TARGETOS` - The target OS for the build (default: `linux`)
  - `TARGETARCH` - The target architecture for the build (default: `amd64`)
  - `CGO_ENABLED` - Enable CGO for the build (default: `0`)
  - `GOPROXY` - The Go module proxy to use (default: `direct`)
  - `PACKAGE` - The package to compile e.g. `./cmd/http` (default: detected via `cmd` directory or `main.go`)

#### Package Detection
  - Find the directory in `cmd` with a `.go` file
  - `main.go` file in the root directory

#### Install Command
`if [ -f go.mod ]; then go mod download; fi`

#### Build Command
`CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /go/bin/app "${PACKAGE}"`

#### Start Command
`["/app/app"]`

---

### Java

[Java](https://www.java.com/) is a class-based, object-oriented programming language that is designed to have as few implementation dependencies as possible.

#### Detected Files
  - `pom.{xml,atom,clj,groovy,rb,scala,yml,yaml}`

#### Version Detection
JDK version:
  - `.tool-versions` - `java {VERSION}`
Maven version:
  - `.tool-versions` - `maven {VERSION}`

#### Runtime Image
`eclipse-temurin:${VERSION}-jdk`

#### Build Args
  - `VERSION` - The version of the JDK to install (default: `17`)
  - `MAVEN_VERSION` - The version of Maven to install (default: `3`)
  - `JAVA_OPTS` - The Java options to pass to the JVM (default: `-Xmx512m -Xms256m`)
  - `BUILD_CMD` - The command to build the project (default: best guess via source code)
  - `START_CMD` - The command to start the project (default: detected via source code)

#### Install Command
- If Maven: `mvn install`

#### Build Command
- If Maven: `mvn -DoutputFile=target/mvn-dependency-list.log -B -DskipTests clean dependency:list install`

#### Start Command
- Default: `java $JAVA_OPTS -jar target/*jar`
- If Spring Boot: `java -Dserver.port=${PORT} $JAVA_OPTS -jar target/*jar`

---

### Next.js

[Next.js](https://nextjs.org/) is a React framework that enables functionality such as server-side rendering and generating static websites.

#### Detected Files
  - `next.config.{js,mjs,cjs,ts,mts}`
  - `next-env.d.ts`
  - `.next/`

#### Version Detection
  - `.tool-versions` - `nodejs {VERSION}`
  - `.nvmrc` - `v{VERSION}`
  - `.node-version` - `v{VERSION}`

#### Runtime Image
`node:${VERSION}-slim`

#### Build Args
  - `VERSION` - The version of Node.js to install (default: `lts`)

#### Install Command
```sh
if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
elif [ -f package-lock.json ]; then npm ci; \
elif [ -f bun.lockb ]; then npm i -g bun && bun install; \
elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm i --frozen-lockfile; \
else echo "Lockfile not found." && exit 1; \
fi
```
#### Build Command
```sh
if [ -f yarn.lock ]; then yarn run build; \
elif [ -f package-lock.json ]; then npm run build; \
elif [ -f bun.lockb ]; then npm i -g bun && bun run build; \
elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm run build; \
else echo "Lockfile not found." && exit 1; \
fi
```

#### Start Command
- If `"output" :"standalone"`  in `next.config.js`: `HOSTNAME="0.0.0.0" node server.js`
- Otherwise `["node_modules/.bin/next", "start", "-H", "0.0.0.0"]`

---

### Node.js

[Node.js](https://nodejs.org/) is a JavaScript runtime built on Chrome's V8 JavaScript engine.

#### Detected Files
  - `yarn.lock`
  - `package-lock.json`
  - `pnpm-lock.yaml`

#### Version Detection
  - `.tool-versions` - `nodejs {VERSION}`
  - `.nvmrc` - `v{VERSION}`
  - `.node-version` - `v{VERSION}`

#### Runtime Image
`node:${VERSION}-slim`

#### Build Args
  - `VERSION` - The version of Node.js to install (default: `lts`)
  - `INSTALL_CMD` - The command to install dependencies (default: detected from source code)
  - `BUILD_CMD` - The command to build the project (default: detected from source code)
  - `START_CMD` - The command to start the project (default: detected from source code)

#### Install Command
- If Yarn: `yarn --frozen-lockfile`
- If npm: `npm ci`
- If pnpm: `corepack enable pnpm && pnpm i --frozen-lockfile`

#### Build Command
In order of precedence:
  - `package.json` scripts: `"build:prod", "build:production", "build-prod", "build-production", "build"`

#### Start Command
In order of precedence:
  - `package.json` scripts: `"serve", "start:prod", "start:production", "start-prod", "start-production", "preview", "start"`
  - `package.json` scripts search for regex matching: `^.*?\b(ts-)?node(mon)?\b.*?(index|main|server|client)\.([cm]?[tj]s)\b`
  - `package.json` main/module file: `node ${mainFile}`

---

### PHP

[PHP](https://www.php.net/) is a popular general-purpose scripting language that is especially suited to web development.

#### Detected Files
  - `composer.json`
  - `index.php`

#### Version Detection
  - `.tool-versions` - `php {VERSION}`
  - `composer.json` - `"php": "{VERSION}"`

#### Runtime Image
`php:${VERSION}-apache`

#### Build Args
  - `VERSION` - The version of PHP to install (default: `8.3`)
  - `INSTALL_CMD` - The command to install dependencies (default: detected via source code)
  - `BUILD_CMD` - The command to build the project (default: detected via source code)
  - `START_CMD` - The command to start the project (default: `apache2-foreground`)

#### Install Command
- If Composer: `composer update && composer install --prefer-dist --no-dev --optimize-autoloader --no-interaction`
- If `package.json` exists: composer install command + see Node.js install command

#### Build Command
- If `package.json` exists: see Node.js build command

#### Start Command
`apache2-foreground`

---

### Python

[Python](https://www.python.org/) is a high-level, interpreted programming language that is known for its readability and simplicity.

#### Detected Files
  - `requirements.txt`
  - `poetry.lock`
  - `Pipefile.lock`
  - `pyproject.toml`
  - `pdm.lock`
  - `main.py`
  - `app.py`
  - `application.py`
  - `app/__init__.py`
  - `filepath.Join(filepath.Base(path), "app.py")`
  - `filepath.Join(filepath.Base(path), "application.py")`
  - `filepath.Join(filepath.Base(path), "main.py")`
  - `filepath.Join(filepath.Base(path), "__init__.py")`

#### Version Detection
  - `.tool-versions` - `python {VERSION}`
  - `.python-version` - `{VERSION}`
  - `runtime.txt` - `python-{VERSION}`

#### Runtime Image
`python:${VERSION}-slim`

#### Build Args
  - `VERSION` - The version of Python to install (default: `3.10`)
  - `INSTALL_CMD` - The command to install dependencies (default: detected from source code)
  - `START_CMD` - The command to start the project (default: detected from source code)

#### Install Command
- If Poetry: `poetry install --no-dev --no-interactive --no-ansi`
- If Pipenv: `PIPENV_VENV_IN_PROJECT=1 pipenv install --deploy`
- If PDM: `pdm install --prod`
- If `pyproject.toml` exists: `pip install --upgrade build setuptools && pip install .`
- If `requirements.txt` exists: `pip install -r requirements.txt`

#### Start Command
- If Django is detected: `python manage.py runserver 0.0.0.0:${PORT}`
- If `pyproject.toml` exists: `python -m ${projectName}`
- Otherwise: `python [main.py, app.py, application.py, app/main.py, app/application.py, app/__init__.py]`

---

### Ruby

[Ruby](https://www.ruby-lang.org/) is a dynamic, open-source programming language with a focus on simplicity and productivity.

#### Detected Files
  - `Gemfile`
  - `Gemfile.lock`
  - `config.ru`
  - `Rakefile`
  - `config/environment.rb`

#### Version Detection
  - `.tool-versions` - `ruby {VERSION}`
  - `.ruby-version` - `{VERSION}`
  - `Gemfile` - `ruby '{VERSION}'`

#### Runtime Image
`ruby:${VERSION}-slim`

#### Build Args
  - `VERSION` - The version of Ruby to install (default: `3.0`)
  - `INSTALL_CMD` - The command to install dependencies (default: detected from source code)
  - `BUILD_CMD` - The command to build the project (default: detected from source code)
  - `START_CMD` - The command to start the project (default: detected from source code)

#### Install Command
- `bundle install`
- If `package.json` exists: `bundle install && [package manager install command]`

#### Build Command
- If Rails: `bundle exec rake assets:precompile`

#### Start Command
- If Rails: `bundle exec rails server -b 0.0.0.0 -p ${PORT}`
- If `config.ru` exists: `bundle exec rackup config.ru -p ${PORT}`
- If `config/environment.rb` exists: `bundle exec rails server -b`
- If `Rakefile` exists: `bundle exec rake`

---

### Rust

[Rust](https://www.rust-lang.org/) is a systems programming language that is known for its speed, memory safety, and parallelism.

#### Detected Files
  - `Cargo.toml`

#### Runtime Image
`debian:stable-slim`

#### Build Args
  - `TARGETOS` - The target OS for the build (default: `linux`)
  - `TARGETARCH` - The target architecture for the build (default: `amd64`)
  - `BIN_NAME` - The name of the release binary (default: detected via `Cargo.toml`)

#### Build Command
```sh 
if [ "${TARGETARCH}" = "amd64" ]; then rustup target add x86_64-unknown-linux-gnu; else rustup target add aarch64-unknown-linux-gnu; fi
if [ "${TARGETARCH}" = "amd64" ]; then cargo zigbuild --release --target x86_64-unknown-linux-gnu; else cargo zigbuild --release --target aarch64-unknown-linux-gnu; fi
```

#### Start Command
Determined by the binary name in the `Cargo.toml` file
- `["/app/app"]`

---

### Static (HTML, CSS, JS)

[Static Web Server](https://static-web-server.net/) is a cross-platform, high-performance & asynchronous web server for static files serving.
It is nearly as fast as Nginx and Lighttpd, but is [easily configurable with environment variables](https://static-web-server.net/configuration/environment-variables/).

#### Detected Files
  - `public/`
  - `static/`
  - `dist/`
  - `index.html`

#### Runtime Image
`joseluisq/static-web-server:${VERSION}-debian`

#### Build Args
  - `VERSION` - The version of the static web server to install (default: `2`)
  - `SERVER_ROOT` - The root directory of the server (default: detected from source code)

---

## Used By

- [FlexStack](https://flexstack.com) - A platform that simplifies the deployment of containerized applications to AWS. 
  FlexStack uses this tool to automatically detect the runtime and framework used by your project, so you can just bring your code and deploy it with confidence.
- *Your project here* - If you're using this tool in your project, let us know! We'd love to feature you here.

## Contributing

Read the [CONTRIBUTING.md](CONTRIBUTING.md) guide to learn how to contribute to this project.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.