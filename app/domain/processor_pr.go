package domain

import (
	"context"
	"fmt"
	"time"
)

// Function indirection for PR creation to keep tests hermetic.
var prCreateFn func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error

// UsePullRequestCreator configures how PRs are created by the PR processor.
func UsePullRequestCreator(f func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error) {
	prCreateFn = f
}

// prProcessor implements the Pull Request flow: clone -> validate -> copy+commit -> create PR.
type prProcessor struct{ ops GitOperations }

func newPRProcessor(ops GitOperations) *prProcessor { return &prProcessor{ops: ops} }

func (p *prProcessor) Process(repo GitRepository, pp ProviderConfig, config Config) error {
	if p.ops == nil {
		return fmt.Errorf("git ops not configured")
	}
	if prCreateFn == nil {
		return fmt.Errorf("pull request creator not configured")
	}

	gitRepo, fs, err := p.ops.CloneGit(repo, pp)
	if err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	valid, err := p.ops.IsValidForBoneClone(gitRepo, config)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	if !valid {
		return nil
	}

	// Generate a branch name for PR head
	branchName := fmt.Sprintf("boneclone/update-%s", time.Now().UTC().Format("20060102-150405"))

	// Copy files, commit, and push to the head branch
	if err := p.ops.CopyFiles(gitRepo, fs, config, pp, branchName); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Create PR from head branch to base target branch
	base := config.Git.TargetBranch
	if base == "" {
		base = "main"
	}
	if err := prCreateFn(context.Background(), pp, repo.Name, base, branchName, nil, ""); err != nil {
		return fmt.Errorf("create PR: %w", err)
	}
	return nil
}
