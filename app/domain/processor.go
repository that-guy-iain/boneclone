package domain

import (
	"fmt"

	billy "github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v6"
)

// Function indirection for testability. Wire these via UseGitOps in main.
var (
	cloneGitFn            func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error)
	isValidForBoneCloneFn func(repo *gogit.Repository, config Config) (bool, error)
	copyFilesFn           func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig) error
)

// UseGitOps configures the git operation functions used by Processor.
func UseGitOps(
	clone func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error),
	validate func(repo *gogit.Repository, config Config) (bool, error),
	copy func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig) error,
) {
	cloneGitFn = clone
	isValidForBoneCloneFn = validate
	copyFilesFn = copy
}

// Processor implements RepoProcessor using wired git operations.
type Processor struct{}

func NewProcessor() *Processor { return &Processor{} }

func (p *Processor) Process(repo GitRepository, pp ProviderConfig, config Config) error {
	fmt.Printf("repo: %s\n", repo.Url)

	gitRepo, fs, err := cloneGitFn(repo, pp)
	if err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	valid, err := isValidForBoneCloneFn(gitRepo, config)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	if valid {
		if err := copyFilesFn(gitRepo, fs, config, pp); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	}

	return nil
}
