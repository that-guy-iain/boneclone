package repository_providers

import (
	"boneclone/app/domain"
	"fmt"
	"strings"
)

func NewProvider(config domain.ProviderConfig) (domain.GitRepositoryProvider, error) {
	if strings.ToLower(config.Provider) == "github" {
		return NewGithubRepositoryProvider(config.Token, config.Org)
	} else if strings.ToLower(config.Provider) == "gitlab" {
		return NewGitlabRepositoryProvider(config.Token, config.Org)
	}

	return nil, fmt.Errorf("unknown provider: %s", config.Provider)
}
