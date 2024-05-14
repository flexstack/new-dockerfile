package runtime

type Runtime interface {
	Name() RuntimeName
	Match(path string) bool
	GenerateDockerfile(path string) ([]byte, error)
}

type RuntimeName string

const (
	RuntimeNameDocker RuntimeName = "Docker" // Done
	RuntimeNameGolang RuntimeName = "Go"     // Done
	RuntimeNameRuby   RuntimeName = "Ruby"   // Done
	RuntimeNamePython RuntimeName = "Python" // Done
	RuntimeNamePHP    RuntimeName = "PHP"    // Done
	RuntimeNameElixir RuntimeName = "Elixir"
	RuntimeNameJava   RuntimeName = "Java"
	RuntimeNameRust   RuntimeName = "Rust"    // Done
	RuntimeNameNextJS RuntimeName = "Next.js" // Done
	RuntimeNameBun    RuntimeName = "Bun"     // Done
	RuntimeNameDeno   RuntimeName = "Deno"    // Done
	RuntimeNameNode   RuntimeName = "Node"    // Done
	RuntimeNameStatic RuntimeName = "Static"  // Done
)
