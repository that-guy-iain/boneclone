package domain

import "testing"

func TestNewProcessorForConfig_SelectsProcessors(t *testing.T) {
	p1 := NewProcessorForConfig(Config{Git: GitConfig{PullRequest: false}})
	if _, ok := p1.(*Processor); !ok {
		t.Fatalf("expected *Processor for PullRequest=false, got %T", p1)
	}

	p2 := NewProcessorForConfig(Config{Git: GitConfig{PullRequest: true}})
	if _, ok := p2.(*prProcessor); !ok {
		t.Fatalf("expected *prProcessor for PullRequest=true, got %T", p2)
	}
}
