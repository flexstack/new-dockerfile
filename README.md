# Autogenerate a Dockerfile

FlexStack's `new-dockerfile` CLI tool and Go package automatically generates a configurable Dockerfile 
based on your project source code. It supports a wide range of languages and frameworks, including Next.js, 
Node.js, Python, Ruby, Java/Spring Boot, Go, Elixir/Phoenix, and more.

For detailed documentation, visit the [FlexStack Documentation](https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile) page.

## Installation

### cURL

```sh
curl -sSL https://flexstack.com/install/new-dockerfile | sh
```

### Go Package

```sh
go get github.com/flexstack/new-dockerfile
```

## Supported Platforms for CLI

- **macOS** (arm64, x86_64)
- **Linux** (arm64, x86_64)
- **Windows** (x86_64, i386)

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