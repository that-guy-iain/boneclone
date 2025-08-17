package domain

import (
	"context"
	"fmt"
	"strings"

	billy "github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v6"
)

const DefaultPRTitle = "BoneClone update"

// skeletonName holds the configured skeleton name for PR messages; defaults to "BoneClone" for backward compatibility.
var skeletonName = "BoneClone"

// SetSkeletonName sets the name used in PR titles/bodies when building default messages.
func SetSkeletonName(name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		skeletonName = name
	}
}

type GitRepositoryProvider interface {
	GetRepositories() (*[]GitRepository, error)
}

// PRBodyBuilder builds the body/description for a pull request given context about the change.
type PRBodyBuilder func(repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) string

// DefaultPRBodyBuilder reproduces the previous PR body format used across providers,
// but uses the configured skeletonName for the intro line.
func DefaultPRBodyBuilder(repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("This is a %s PR.\n\n", skeletonName))
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

type GitOperations interface {
	CloneGit(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error)
	IsValidForBoneClone(repo *gogit.Repository, config Config) (bool, error)
	CopyFiles(repo *gogit.Repository, fs billy.Filesystem, config Config, provider ProviderConfig, targetBranch string) error
}

type GitRepository struct {
	Name string
	Url  string
}
