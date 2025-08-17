package domain

import (
	"fmt"
)

// prProcessor is a placeholder implementation that will later create pull requests.
// For now, it simply indicates the PR flow is not yet implemented.
// It still clones and validates to keep parity with push flow structure in future extensions.
type prProcessor struct{}

func newPRProcessor() *prProcessor { return &prProcessor{} }

func (p *prProcessor) Process(repo GitRepository, pp ProviderConfig, config Config) error {
	// Placeholder: later this will create a branch, push changes to it, and open a PR.
	return fmt.Errorf("pull request flow not implemented yet")
}
