package domain

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
)

type fakeProvider struct {
	repos *[]GitRepository
	err   error
}

func (f *fakeProvider) GetRepositories() (*[]GitRepository, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.repos, nil
}

type recordingProcessor struct {
	mu    sync.Mutex
	calls []procCall
}

type procCall struct {
	repo     GitRepository
	provider ProviderConfig
	config   Config
}

func (p *recordingProcessor) Process(repo GitRepository, provider ProviderConfig, config Config) error {
	p.mu.Lock()
	p.calls = append(p.calls, procCall{repo: repo, provider: provider, config: config})
	p.mu.Unlock()
	return nil
}

// helper to build a ProviderFactory that interprets ProviderConfig.Provider values
//
//	"ok": returns two repos named org-1, org-2
//	"fail": factory returns error
//	"errlist": provider returns error on GetRepositories
func testFactory() ProviderFactory {
	return func(pc ProviderConfig) (GitRepositoryProvider, error) {
		switch pc.Provider {
		case "fail":
			return nil, errors.New("factory failure")
		case "errlist":
			return &fakeProvider{err: errors.New("list failure")}, nil
		case "ok":
			repos := []GitRepository{
				{Name: pc.Org + "-1", Url: "https://example.com/" + pc.Org + "/1"},
				{Name: pc.Org + "-2", Url: "https://example.com/" + pc.Org + "/2"},
			}
			return &fakeProvider{repos: &repos}, nil
		default:
			// default to no repos
			repos := []GitRepository{}
			return &fakeProvider{repos: &repos}, nil
		}
	}
}

func TestRun_ProcessesAllRepos(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{
			{Provider: "ok", Org: "one"},
			{Provider: "ok", Org: "two"},
		},
	}

	rp := &recordingProcessor{}

	if err := Run(context.Background(), cfg, testFactory(), rp); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Expect 4 calls (2 repos per provider)
	if len(rp.calls) != 4 {
		t.Fatalf("expected 4 Process calls, got %d", len(rp.calls))
	}

	// Verify the set of repo names processed
	gotNames := make([]string, 0, len(rp.calls))
	for _, c := range rp.calls {
		// Ensure full config is passed through
		if c.config.Providers == nil || len(c.config.Providers) != 2 {
			t.Fatalf("unexpected config passed to processor: %+v", c.config)
		}
		gotNames = append(gotNames, c.repo.Name)
	}
	sort.Strings(gotNames)
	want := []string{"one-1", "one-2", "two-1", "two-2"}
	for i := range want {
		if gotNames[i] != want[i] {
			t.Fatalf("unexpected repo set processed. got=%v want=%v", gotNames, want)
		}
	}
}

func TestRun_SkipsOnFactoryError(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{
			{Provider: "fail", Org: "ignored"},
			{Provider: "ok", Org: "good"},
		},
	}

	rp := &recordingProcessor{}

	if err := Run(context.Background(), cfg, testFactory(), rp); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Only the "ok" provider should have been processed: 2 repos
	if len(rp.calls) != 2 {
		t.Fatalf("expected 2 Process calls after factory error skip, got %d", len(rp.calls))
	}

	names := []string{rp.calls[0].repo.Name, rp.calls[1].repo.Name}
	sort.Strings(names)
	if names[0] != "good-1" || names[1] != "good-2" {
		t.Fatalf("unexpected repos processed: %v", names)
	}
}

func TestRun_SkipsOnListError(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{
			{Provider: "errlist", Org: "bad"},
			{Provider: "ok", Org: "good"},
		},
	}

	rp := &recordingProcessor{}

	if err := Run(context.Background(), cfg, testFactory(), rp); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Only the "ok" provider should have been processed: 2 repos
	if len(rp.calls) != 2 {
		t.Fatalf("expected 2 Process calls after list error skip, got %d", len(rp.calls))
	}

	for _, c := range rp.calls {
		if c.provider.Org != "good" {
			t.Fatalf("expected provider Org 'good', got %q", c.provider.Org)
		}
	}
}
