package domain

import (
	"context"
	"errors"
	"strings"
	"testing"

	billy "github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v6"
)

// fakeOpsPR implements GitOperations for prProcessor tests and captures inputs.
type fakeOpsPR struct {
	cloneErr   error
	valid      bool
	validErr   error
	copyErr    error
	copyCalled bool
	lastBranch string
}

func (f *fakeOpsPR) CloneGit(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
	return nil, nil, f.cloneErr
}

func (f *fakeOpsPR) IsValidForBoneClone(repo *gogit.Repository, cfg Config) (bool, RemoteConfig, error) {
	return f.valid, RemoteConfig{}, f.validErr
}

func (f *fakeOpsPR) CopyFiles(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error {
	f.copyCalled = true
	f.lastBranch = targetBranch
	return f.copyErr
}

// fake PR provider/manager implements both discovery and PR creation interfaces.
type fakePRProviderManager struct {
	called      bool
	repo        string
	base        string
	head        string
}

func (f *fakePRProviderManager) GetRepositories() (*[]GitRepository, error) { return &[]GitRepository{}, nil }

func (f *fakePRProviderManager) CreatePullRequest(_ context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody PRBodyBuilder) (PRInfo, error) {
	f.called = true
	f.repo = repo
	f.base = baseBranch
	f.head = headBranch
	return PRInfo{ID: 1, URL: "http://example/pr/1"}, nil
}

func (f *fakePRProviderManager) AssignReviewers(_ context.Context, _ string, _ PRInfo, _ []string) error { return nil }

func TestPRProcessor_ErrWhenOpsNil(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }
	p := newPRProcessor(nil, pf)

	repo := GitRepository{Name: "repo1"}
	pp := ProviderConfig{}
	cfg := Config{}

	if err := p.Process(repo, pp, cfg); err == nil || !strings.Contains(err.Error(), "git ops not configured") {
		t.Fatalf("expected git ops not configured error, got %v", err)
	}
}

func TestPRProcessor_ErrWhenProviderFactoryNil(t *testing.T) {
	ops := &fakeOpsPR{}
	p := newPRProcessor(ops, nil)

	repo := GitRepository{Name: "r"}
	pp := ProviderConfig{}
	cfg := Config{}

	if err := p.Process(repo, pp, cfg); err == nil || !strings.Contains(err.Error(), "provider factory not configured") {
		t.Fatalf("expected provider factory not configured error, got %v", err)
	}
}

func TestPRProcessor_CloneAndValidateErrors(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }

	// Clone error
	ops := &fakeOpsPR{cloneErr: errors.New("boom")}
	p := newPRProcessor(ops, pf)
	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "clone: boom" {
		t.Fatalf("expected clone error wrapping, got %v", err)
	}

	// Validate error
	ops = &fakeOpsPR{validErr: errors.New("valerr")}
	p = newPRProcessor(ops, pf)
	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "validate: valerr" {
		t.Fatalf("expected validate error wrapping, got %v", err)
	}
}

func TestPRProcessor_NotValid_SkipsCopyAndPR(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }
	ops := &fakeOpsPR{valid: false}
	p := newPRProcessor(ops, pf)

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ops.copyCalled {
		t.Fatalf("expected CopyFiles not to be called when not valid")
	}
	if fakeProv.called {
		t.Fatalf("expected PR creation not to be called when not valid")
	}
}

func TestPRProcessor_CopyError(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }
	ops := &fakeOpsPR{valid: true, copyErr: errors.New("cperr")}
	p := newPRProcessor(ops, pf)

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "copy: cperr" {
		t.Fatalf("expected copy error wrapping, got %v", err)
	}
	if fakeProv.called {
		t.Fatalf("expected PR creation not to be called on copy error")
	}
}

func TestPRProcessor_Success_UsesTargetBranch_AndCallsPRCreator(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }
	ops := &fakeOpsPR{valid: true}
	p := newPRProcessor(ops, pf)

	repo := GitRepository{Name: "my-repo"}
	cfg := Config{Git: GitConfig{TargetBranch: "develop"}}

	if err := p.Process(repo, ProviderConfig{}, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ops.copyCalled {
		t.Fatalf("expected CopyFiles to be called")
	}
	if !strings.HasPrefix(ops.lastBranch, "boneclone/update-") {
		t.Fatalf("expected branch to start with boneclone/update-, got %q", ops.lastBranch)
	}
	if !fakeProv.called {
		t.Fatalf("expected PR creator to be called")
	}
	if fakeProv.repo != repo.Name {
		t.Fatalf("expected repo name %q, got %q", repo.Name, fakeProv.repo)
	}
	if fakeProv.base != "develop" {
		t.Fatalf("expected base branch 'develop', got %q", fakeProv.base)
	}
	if fakeProv.head != ops.lastBranch {
		t.Fatalf("expected head branch %q to match CopyFiles branch %q", fakeProv.head, ops.lastBranch)
	}
}

func TestPRProcessor_Success_DefaultsBaseToMain(t *testing.T) {
	fakeProv := &fakePRProviderManager{}
	pf := func(pp ProviderConfig) (GitRepositoryProvider, error) { return fakeProv, nil }
	ops := &fakeOpsPR{valid: true}
	p := newPRProcessor(ops, pf)

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fakeProv.base != "main" {
		t.Fatalf("expected default base 'main', got %q", fakeProv.base)
	}
}
