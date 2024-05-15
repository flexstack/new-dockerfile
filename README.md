# Autogenerate a Dockerfile

`new-dockerfile` is a CLI tool and Go package automatically generates a configurable Dockerfile 
based on your project source code. It supports a wide range of languages and frameworks, including Next.js, 
Node.js, Python, Ruby, Java/Spring Boot, Go, Elixir/Phoenix, and more.

See the [FlexStack Documentation](https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile) page for FlexStack-specific documentation
related to this tool.

## Features

- [x] Automatically detect the runtime and framework used by your project
- [x] Use version managers like [asdf](https://github.com/asdf-vm), nvm, rbenv, and pyenv to install the correct version of the runtime
- [x] Make a best effort to detect any install, build, and run commands
- [x] Generate a Dockerfile with sensible defaults that are configurable via [Docker Build Args](https://docs.docker.com/build/guide/build-args/)
- [x] Support for a wide range of the most popular languages and frameworks including Next.js, Phoenix, Spring Boot, Django, and more
- [x] Use Debian Slim as the runtime image for a smaller image size and better security, while still supporting the most common dependencies and avoiding deployment headaches caused by Alpine Linux gotchas
- [x] Includes `wget` in the runtime image for adding health checks to services, e.g. `wget -nv -t1 --spider 'http://localhost:8080/healthz' || exit 1`
- [x] Use multi-stage builds to reduce the size of the final image
- [x] Supports multi-platform images that run on both x86 and ARM CPU architectures

## Supported Runtimes

- [Bun](#bun)
- [Deno](#deno))
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

## How it Works

The tool searches for common files and directories in your project to determine the runtime and framework.
For example, if it finds a `package.json` file, it will assume the project is a Node.js project unless
a `next.config.js` file is present, in which case it will assume the project is a Next.js project.

From there, it will read any `.tool-versions` or other version manager files to determine the correct version
of the runtime to install. It will then make a best effort to detect any install, build, and run commands.
For example, a `serve`, `start`, `start:prod` command in a `package.json` file will be used as the run command.

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
  - `package.json` scripts: `"serve", "start:prod", "start:production", "start-prod", "start-production", "start"`
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
  - `deno.jsonc` tasks: `"serve", "start:prod", "start:production", "start-prod", "start-production", "start"`
  - Main/module file: `deno run ["mod.ts", "src/mod.ts", "main.ts", "src/main.ts", "index.ts", "src/index.ts]"`
  
---

### Elixir

---

### Go

---

### Java

---

### Next.js

---

### Node.js

---

### PHP

---

### Python

---

### Ruby

---

### Rust

---

### Static (HTML, CSS, JS)

---

## Contributing

Read the [CONTRIBUTING.md](CONTRIBUTING.md) guide to learn how to contribute to this project.

## Used By

- [FlexStack](https://flexstack.com) - A platform that simplifies the deployment of containerized applications to AWS. 
  FlexStack uses this tool to automatically detect the runtime and framework used by your project, so you can just bring your code and deploy it with confidence.
- *Your project here* - If you're using this tool in your project, let us know! We'd love to feature you here.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.