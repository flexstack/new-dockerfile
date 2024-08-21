package runtime

// An interface that all runtimes must implement.
type Runtime interface {
	// Returns the name of the runtime.
	Name() RuntimeName
	// Returns true if the runtime can be used for the given path.
	Match(path string) bool
	// Generates a Dockerfile for the given path.
	GenerateDockerfile(path string, data ...map[string]string) ([]byte, error)
}

type RuntimeName string

const (
	RuntimeNameGolang RuntimeName = "Go"
	RuntimeNameRuby   RuntimeName = "Ruby"
	RuntimeNamePython RuntimeName = "Python"
	RuntimeNamePHP    RuntimeName = "PHP"
	RuntimeNameElixir RuntimeName = "Elixir"
	RuntimeNameJava   RuntimeName = "Java"
	RuntimeNameRust   RuntimeName = "Rust"
	RuntimeNameNextJS RuntimeName = "Next.js"
	RuntimeNameBun    RuntimeName = "Bun"
	RuntimeNameDeno   RuntimeName = "Deno"
	RuntimeNameNode   RuntimeName = "Node"
	RuntimeNameStatic RuntimeName = "Static"
)
