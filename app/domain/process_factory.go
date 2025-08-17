package domain

// NewProcessorForConfig returns a RepoProcessor implementation based on configuration.
// When cfg.Git.PullRequest is true, a PR flow implementation is returned (placeholder for now).
// Otherwise, it returns the default push-to-target-branch processor.
func NewProcessorForConfig(cfg Config) RepoProcessor {
	if cfg.Git.PullRequest {
		return newPRProcessor()
	}
	return NewProcessor()
}
