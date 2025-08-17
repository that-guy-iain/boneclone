package domain

// NewProcessorForConfig returns a RepoProcessor implementation based on configuration.
// When cfg.Git.PullRequest is true, a PR flow implementation is returned.
// Otherwise, it returns the default push-to-target-branch processor.
func NewProcessorForConfig(cfg Config, ops GitOperations, pf ProviderFactory) RepoProcessor {
	if cfg.Git.PullRequest {
		return newPRProcessor(ops, pf)
	}
	return NewProcessor(ops)
}
