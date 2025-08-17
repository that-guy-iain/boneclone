package domain

import (
	"context"
	"fmt"
	"time"
)

// prProcessor implements the Pull Request flow: clone -> validate -> copy+commit -> create PR.
type prProcessor struct{
	ops         GitOperations
	newProvider ProviderFactory
}

func newPRProcessor(ops GitOperations, pf ProviderFactory) *prProcessor { return &prProcessor{ops: ops, newProvider: pf} }

func (p *prProcessor) Process(repo GitRepository, pp ProviderConfig, config Config) error {
	if p.ops == nil {
		return fmt.Errorf("git ops not configured")
	}
	if p.newProvider == nil {
		return fmt.Errorf("provider factory not configured")
	}

	gitRepo, fs, err := p.ops.CloneGit(repo, pp)
	if err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	valid, remoteCfg, err := p.ops.IsValidForBoneClone(gitRepo, config)
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

	// Build PR title (use configured name when present)
	prTitle := DefaultPRTitle
	if name := config.Identifier.Name; name != "" {
		prTitle = fmt.Sprintf("%s update", name)
	}

	prov, err := p.newProvider(pp)
	if err != nil {
		return fmt.Errorf("create PR: %w", err)
	}
	if prMgr, ok := prov.(PullRequestManager); ok {
		pr, err := prMgr.CreatePullRequest(context.Background(), repo.Name, base, branchName, prTitle, nil, "", DefaultPRBodyBuilder)
		if err != nil {
			return fmt.Errorf("create PR: %w", err)
		}
		// Attempt to assign reviewers from remote config; failures are ignored (silent)
		for _, r := range remoteCfg.Reviewers {
			_ = prMgr.AssignReviewers(context.Background(), repo.Name, pr, []string{r})
		}
		return nil
	}

	return fmt.Errorf("provider %s does not support pull requests", pp.Provider)
}
