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
func (g GithubRepositoryProvider) CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody domain.PRBodyBuilder) (domain.PRInfo, error) {
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

	pr, _, err := g.github.PullRequests.Create(ctx, g.orgName, repo, newPR)
	if err != nil {
		return domain.PRInfo{}, err
	}
	id := 0
	if pr.Number != nil { id = *pr.Number }
	url := ""
	if pr.HTMLURL != nil { url = *pr.HTMLURL }
	return domain.PRInfo{ID: id, URL: url}, nil
}

// AssignReviewers requests reviewers on an existing PR. Errors are returned to the caller to decide handling.
func (g GithubRepositoryProvider) AssignReviewers(ctx context.Context, repo string, pr domain.PRInfo, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	req := github.ReviewersRequest{Reviewers: reviewers}
	_, _, err := g.github.PullRequests.RequestReviewers(ctx, g.orgName, repo, pr.ID, req)
	return err
}

func NewGithubRepositoryProvider(token, orgName string) (domain.GitRepositoryProvider, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	return &GithubRepositoryProvider{client, orgName}, nil
}
