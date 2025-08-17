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

func (f *fakeOpsPR) IsValidForBoneClone(repo *gogit.Repository, cfg Config) (bool, error) {
	return f.valid, f.validErr
}

func (f *fakeOpsPR) CopyFiles(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error {
	f.copyCalled = true
	f.lastBranch = targetBranch
	return f.copyErr
}

func TestPRProcessor_ErrWhenOpsNil(t *testing.T) {
	p := newPRProcessor(nil)
	// Ensure pr creator is set so we fail specifically on ops nil
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error { return nil })
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	repo := GitRepository{Name: "repo1"}
	pp := ProviderConfig{}
	cfg := Config{}

	if err := p.Process(repo, pp, cfg); err == nil || !strings.Contains(err.Error(), "git ops not configured") {
		t.Fatalf("expected git ops not configured error, got %v", err)
	}
}

func TestPRProcessor_ErrWhenPRCreatorNil(t *testing.T) {
	ops := &fakeOpsPR{}
	p := newPRProcessor(ops)
	// Explicitly unset PR creator
	UsePullRequestCreator(nil)
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	repo := GitRepository{Name: "r"}
	pp := ProviderConfig{}
	cfg := Config{}

	if err := p.Process(repo, pp, cfg); err == nil || !strings.Contains(err.Error(), "pull request creator not configured") {
		t.Fatalf("expected pr creator not configured error, got %v", err)
	}
}

func TestPRProcessor_CloneAndValidateErrors(t *testing.T) {
	// Clone error
	ops := &fakeOpsPR{cloneErr: errors.New("boom")}
	p := newPRProcessor(ops)
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error { return nil })
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "clone: boom" {
		t.Fatalf("expected clone error wrapping, got %v", err)
	}

	// Validate error
	ops = &fakeOpsPR{validErr: errors.New("valerr")}
	p = newPRProcessor(ops)
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error { return nil })
	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "validate: valerr" {
		t.Fatalf("expected validate error wrapping, got %v", err)
	}
}

func TestPRProcessor_NotValid_SkipsCopyAndPR(t *testing.T) {
	ops := &fakeOpsPR{valid: false}
	p := newPRProcessor(ops)
	prCalled := false
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error {
		prCalled = true
		return nil
	})
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ops.copyCalled {
		t.Fatalf("expected CopyFiles not to be called when not valid")
	}
	if prCalled {
		t.Fatalf("expected PR creator not to be called when not valid")
	}
}

func TestPRProcessor_CopyError(t *testing.T) {
	ops := &fakeOpsPR{valid: true, copyErr: errors.New("cperr")}
	p := newPRProcessor(ops)
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error { return nil })
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err == nil || err.Error() != "copy: cperr" {
		t.Fatalf("expected copy error wrapping, got %v", err)
	}
}

func TestPRProcessor_Success_UsesTargetBranch_AndCallsPRCreator(t *testing.T) {
	ops := &fakeOpsPR{valid: true}
	p := newPRProcessor(ops)

	called := false
	var gotRepo, gotBase, gotHead string
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error {
		called = true
		gotRepo, gotBase, gotHead = repo, baseBranch, headBranch
		return nil
	})
	t.Cleanup(func() { UsePullRequestCreator(nil) })

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
	if !called {
		t.Fatalf("expected PR creator to be called")
	}
	if gotRepo != repo.Name {
		t.Fatalf("expected repo name %q, got %q", repo.Name, gotRepo)
	}
	if gotBase != "develop" {
		t.Fatalf("expected base branch 'develop', got %q", gotBase)
	}
	if gotHead != ops.lastBranch {
		t.Fatalf("expected head branch %q to match CopyFiles branch %q", gotHead, ops.lastBranch)
	}
}

func TestPRProcessor_Success_DefaultsBaseToMain(t *testing.T) {
	ops := &fakeOpsPR{valid: true}
	p := newPRProcessor(ops)

	var gotBase string
	UsePullRequestCreator(func(ctx context.Context, pp ProviderConfig, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error {
		gotBase = baseBranch
		return nil
	})
	t.Cleanup(func() { UsePullRequestCreator(nil) })

	if err := p.Process(GitRepository{Name: "r"}, ProviderConfig{}, Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBase != "main" {
		t.Fatalf("expected default base 'main', got %q", gotBase)
	}
}
