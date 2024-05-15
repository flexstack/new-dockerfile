# Contribute

> A guide for contributing to `new-dockerfile`

## Getting Started

1. [Install asdf](https://asdf-vm.com/guide/getting-started.html) - A CLI tool that can manage multiple
   language runtime versions on a per-project basis

3. Clone the repo and open in VSCode

```sh
git clone https://github.com/flexstack/new-dockerfile
code new-dockerfile
```

4. Install the project's dependencies

```sh
asdf install
go mod download
```

## Development

Run the CLI in development mode:

```sh
go run ./cmd/new-dockerfile --help
```

Vet code
```sh
go vet ./...
```

Run tests
```sh
go test -v ./...
```

## Open an issue

If you find a bug or want to request a new feature, please open an issue.

## Submit a pull request

Before submitting a feature pull request, it is important to open an issue to discuss what you plan to work on to ensure success in releasing your changes.
For small bug fixes or improvements, go ahead and submit a pull request without an issue.

1. Fork the repo
2. Create a new branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Commit your changes (`git commit -am 'Add my feature'`)
5. Push to the branch (`git push origin feature/my-feature`)
6. Create a new Pull Request

## License

MIT