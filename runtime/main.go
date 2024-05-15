package runtime

type Runtime interface {
	Name() RuntimeName
	Match(path string) bool
	GenerateDockerfile(path string) ([]byte, error)
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
