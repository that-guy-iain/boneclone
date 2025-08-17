package repository_providers

import (
	"fmt"
	"strings"

	"go.iain.rocks/boneclone/app/domain"
)

func NewProvider(config domain.ProviderConfig) (domain.GitRepositoryProvider, error) {
	switch strings.ToLower(config.Provider) {
	case "github":
		return NewGithubRepositoryProvider(config.Token, config.Org)
	case "gitlab":
		return NewGitlabRepositoryProvider(config.Token, config.Org)
	case "azure":
		return NewAzureRepositoryProvider(config.Token, config.Org)
	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
}
