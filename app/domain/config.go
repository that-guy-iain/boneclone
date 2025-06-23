package domain

type Config struct {
	Providers  []ProviderConfig `koanf:"providers"`
	Files      FileConfig       `koanf:"files"`
	Identifier IdentifierConfig `koanf:"identifier"`
}

type ProviderConfig struct {
	Provider string `koanf:"provider"`
	Username string `koanf:"username"`
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
