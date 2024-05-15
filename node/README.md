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

- Bun
- Deno
- Docker
- Elixir
- Go
- Java
- Next.js
- Node.js
- PHP
- Python
- Ruby
- Rust
- Static (HTML, CSS, JS)

## Usage

Using `npx`:

```sh
npx new-dockerfile [options]
```

Install the CLI globally:

```sh
npm install -g new-dockerfile
```

## Options

- `--path` - Path to the project source code (default: `.`)
- `--write` - Write the generated Dockerfile to the project at the specified path (default: `false`)
- `--runtime` - Force a specific runtime, e.g. `node` (default: `auto`)
- `--quiet` - Disable all logging except for errors (default: `false`)
- `--help` - Show help

## Examples

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

## Used By

- [FlexStack](https://flexstack.com) - A platform that simplifies the deployment of containerized applications to AWS. 
  FlexStack uses this tool to automatically detect the runtime and framework used by your project, so you can just bring your code and deploy it with confidence.
- *Your project here* - If you're using this tool in your project, let us know! We'd love to feature you here.
