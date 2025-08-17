package repository_providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v72/github"

	"go.iain.rocks/boneclone/app/domain"
)

type GithubRepositoryProvider struct {
	github  *github.Client
	orgName string
}

func (g GithubRepositoryProvider) GetRepositories() (*[]domain.GitRepository, error) {
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

	return &output, nil
}

// CreatePullRequest creates a PR on the specified repository within the configured org.
// The PR body indicates it's a BoneClone PR, lists changed files, and records the original author.
func (g GithubRepositoryProvider) CreatePullRequest(ctx context.Context, repo string, baseBranch string, headBranch string, filesChanged []string, originalAuthor string) error {
	title := "BoneClone update"
	var bodyBuilder strings.Builder
	bodyBuilder.WriteString("This is a BoneClone PR.\n\n")
	if originalAuthor != "" {
		bodyBuilder.WriteString(fmt.Sprintf("Original author: %s\n\n", originalAuthor))
	}
	if len(filesChanged) > 0 {
		bodyBuilder.WriteString("Files changed:\n")
		for _, f := range filesChanged {
			bodyBuilder.WriteString("- ")
			bodyBuilder.WriteString(f)
			bodyBuilder.WriteString("\n")
		}
	}
	body := bodyBuilder.String()

	newPR := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(headBranch),
		Base:  github.String(baseBranch),
		Body:  github.String(body),
	}

	_, _, err := g.github.PullRequests.Create(ctx, g.orgName, repo, newPR)
	return err
}

func NewGithubRepositoryProvider(token, orgName string) (domain.GitRepositoryProvider, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	return &GithubRepositoryProvider{client, orgName}, nil
}
