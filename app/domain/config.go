package domain

type Config struct {
	Providers  []ProviderConfig `koanf:"providers"`
	Files      FileConfig       `koanf:"files"`
	Identifier IdentifierConfig `koanf:"identifier"`
	Git        GitConfig        `koanf:"git"`
}

type ProviderConfig struct {
	Provider string `koanf:"provider"`
	Username string `koanf:"username"`
	Org      string `koanf:"org"`
	Token    string `koanf:"token"`
}

type FileConfig struct {
	Include []string `koanf:"include"`
	Exclude []string `koanf:"exclude"`
}

type IdentifierConfig struct {
	Filename string `koanf:"filename"`
	Content  string `koanf:"content"`
}

type GitConfig struct {
	Name         string `koanf:"name"`
	Email        string `koanf:"email"`
	PullRequest  bool   `koanf:"pullRequest"`
}
