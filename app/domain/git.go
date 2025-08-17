package domain

import (
	"context"
	"fmt"
	"strings"
)

const DefaultPRTitle = "BoneClone update"

type GitRepositoryProvider interface {
	GetRepositories() (*[]GitRepository, error)
}

// PRBodyBuilder builds the body/description for a pull request given context about the change.
type PRBodyBuilder func(repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) string

// DefaultPRBodyBuilder reproduces the previous PR body format used across providers.
func DefaultPRBodyBuilder(repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) string {
	var b strings.Builder
	b.WriteString("This is a BoneClone PR.\n\n")
	if originalAuthor != "" {
		b.WriteString(fmt.Sprintf("Original author: %s\n\n", originalAuthor))
	}
	if len(filesChanged) > 0 {
		b.WriteString("Files changed:\n")
		for _, f := range filesChanged {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
	}
	return b.String()
}

type PullRequestManager interface {
	CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody PRBodyBuilder) error
}

type GitRepository struct {
	Name string
	Url  string
}
