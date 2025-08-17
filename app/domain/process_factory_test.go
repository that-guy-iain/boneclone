package domain_test

import (
	"testing"

	"go.iain.rocks/boneclone/app/domain"
)

func TestNewProcessorForConfig_SelectsProcessors(t *testing.T) {
	p1 := domain.NewProcessorForConfig(domain.Config{Git: domain.GitConfig{PullRequest: false}})
	if _, ok := p1.(*domain.Processor); !ok {
		t.Fatalf("expected *domain.Processor for PullRequest=false, got %T", p1)
	}

	p2 := domain.NewProcessorForConfig(domain.Config{Git: domain.GitConfig{PullRequest: true}})
	// prProcessor is unexported; assert behavior via Process()
	if err := p2.Process(domain.GitRepository{}, domain.ProviderConfig{}, domain.Config{}); err == nil || err.Error() != "git ops not configured" {
		t.Fatalf("expected PR processor to return 'git ops not configured', got: %v", err)
	}
}
