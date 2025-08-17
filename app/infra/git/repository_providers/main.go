package repository_providers

import (
	"fmt"
	"go.iain.rocks/boneclone/app/domain"
	"strings"
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
