package repository_providers

import (
	"boneclone/app/domain"
	"context"
)
import "github.com/google/go-github/v72/github"

type GithubRepositoryProvider struct {
	github  *github.Client
	orgName string
}

func (g GithubRepositoryProvider) GetRepositories() ([]domain.GitRepository, error) {
	repos, _, err := g.github.Repositories.ListByOrg(context.Background(), g.orgName, nil)

	if err != nil {
		return nil, err
	}
	output := []domain.GitRepository{}
	for _, repo := range repos {
		output = append(output, domain.GitRepository{
			Name: *repo.Name,
			Url:  *repo.CloneURL,
		})
	}

	return output, nil
}

func NewGithubRepositoryProvider(token string, orgName string) (domain.GitRepositoryProvider, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	return &GithubRepositoryProvider{client, orgName}, nil
}
