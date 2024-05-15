# Autogenerate a Dockerfile from your project source code

FlexStack's `new-dockerfile` CLI tool automatically generates a configurable Dockerfile based on your project source code. 
It supports a wide range of languages and frameworks, including Next.js, Node.js, Python, Ruby, Java/Spring Boot, Go, 
Elixir/Phoenix, and more.

For detailed documentation, visit the [FlexStack Documentation](https://flexstack.com/docs/languages-and-frameworks/autogenerate-dockerfile) page.

## Supported Platforms

- **macOS** (arm64, x86_64)
- **Linux** (arm64, x86_64)
- **Windows** (x86_64, i386)

## Usage

```bash
npx new-dockerfile [options]
```

## Options

- `--path` - Path to the project source code (default: `.`)
- `--write` - Write the generated Dockerfile to the project at the specified path (default: `false`)
- `--runtime` - Force a specific runtime, e.g. `node` (default: `auto`)
- `--quiet` - Disable all logging except for errors (default: `false`)
- `--help` - Show help