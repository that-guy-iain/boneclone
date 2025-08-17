package domain

import (
	"fmt"
)

// Processor implements RepoProcessor using injected git operations.
type Processor struct{ ops GitOperations }

func NewProcessor(ops GitOperations) *Processor { return &Processor{ops: ops} }

func (p *Processor) Process(repo GitRepository, pp ProviderConfig, config Config) error {
	if p.ops == nil {
		return fmt.Errorf("git ops not configured")
	}
	fmt.Printf("repo: %s\n", repo.Url)

	gitRepo, fs, err := p.ops.CloneGit(repo, pp)
	if err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	valid, _, err := p.ops.IsValidForBoneClone(gitRepo, config)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	if valid {
		tb := config.Git.TargetBranch
		if err := p.ops.CopyFiles(gitRepo, fs, config, pp, tb); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	}

	return nil
}
