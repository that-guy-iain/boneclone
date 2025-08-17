package domain

import "context"

type GitRepositoryProvider interface {
	GetRepositories() (*[]GitRepository, error)
}

type PullRequestManager interface {
	CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error
}

type GitRepository struct {
	Name string
	Url  string
}
