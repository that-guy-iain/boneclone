package domain

type Config struct {
	Providers []ProviderConfig `koanf:"providers"`
}

type ProviderConfig struct {
	Provider string `koanf:"provider"`
	Username string `koanf:"username"`
	Token    string `koanf:"token"`
}
