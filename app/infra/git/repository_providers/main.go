package repository_providers

import (
	"fmt"
	"strings"

	"go.iain.rocks/boneclone/app/domain"
)

func NewProvider(config domain.ProviderConfig) (domain.GitRepositoryProvider, error) {
	if strings.ToLower(config.Provider) == "github" {
		return NewGithubRepositoryProvider(config.Token, config.Org)
	} else if strings.ToLower(config.Provider) == "gitlab" {
		return NewGitlabRepositoryProvider(config.Token, config.Org)
	} else if strings.ToLower(config.Provider) == "azure" {
		return NewAzureRepositoryProvider(config.Token, config.Org)
	}

	return nil, fmt.Errorf("unknown provider: %s", config.Provider)
}
