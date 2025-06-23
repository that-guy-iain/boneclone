package repository_providers

import (
	"fmt"
	"strings"
	"superspreader/app/domain"
)

func NewProvider(config domain.ProviderConfig) (domain.GitRepositoryProvider, error) {
	if strings.ToLower(config.Provider) == "github" {
		return NewGithubRepositoryProvider(config.Token, config.Username)
	}

	return nil, fmt.Errorf("unknown provider: %s", config.Provider)
}
