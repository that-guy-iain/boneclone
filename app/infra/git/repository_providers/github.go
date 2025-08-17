package repository_providers

import (
	"context"

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
// The PR body is produced by the provided buildBody function.
func (g GithubRepositoryProvider) CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody domain.PRBodyBuilder) error {
	body := ""
	if buildBody != nil {
		body = buildBody(repo, baseBranch, headBranch, filesChanged, originalAuthor)
	}

	newPR := &github.NewPullRequest{
		Title: github.Ptr(title),
		Head:  github.Ptr(headBranch),
		Base:  github.Ptr(baseBranch),
		Body:  github.Ptr(body),
	}

	_, _, err := g.github.PullRequests.Create(ctx, g.orgName, repo, newPR)
	return err
}

func NewGithubRepositoryProvider(token, orgName string) (domain.GitRepositoryProvider, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	return &GithubRepositoryProvider{client, orgName}, nil
}
