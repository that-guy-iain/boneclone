package domain

import (
	"errors"
	"testing"

	billy "github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v6"
)

func withStubbedFns(t *testing.T, stub func()) {
	t.Helper()
	origClone := cloneGitFn
	origValid := isValidForBoneCloneFn
	origCopy := copyFilesFn
	defer func() {
		cloneGitFn = origClone
		isValidForBoneCloneFn = origValid
		copyFilesFn = origCopy
	}()
	stub()
}

func TestProcessor_Process_CloneError(t *testing.T) {
	p := NewProcessor()
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}

	withStubbedFns(t, func() {
		cloneGitFn = func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
			return nil, nil, errors.New("boom")
		}
		// Not expected to be called
		isValidForBoneCloneFn = func(repo *gogit.Repository, config Config) (bool, error) { return false, nil }
		copyFilesFn = func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error { return nil }

		err := p.Process(repo, pp, cfg)
		if err == nil || err.Error() != "clone: boom" {
			t.Fatalf("expected clone error wrapping, got: %v", err)
		}
	})
}

func TestProcessor_Process_ValidateError(t *testing.T) {
	p := NewProcessor()
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}

	withStubbedFns(t, func() {
		cloneGitFn = func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
			return nil, nil, nil
		}
		isValidForBoneCloneFn = func(repo *gogit.Repository, config Config) (bool, error) {
			return false, errors.New("valerr")
		}
  copyFilesFn = func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error { return nil }

		err := p.Process(repo, pp, cfg)
		if err == nil || err.Error() != "validate: valerr" {
			t.Fatalf("expected validate error wrapping, got: %v", err)
		}
	})
}

func TestProcessor_Process_NotValid_SkipsCopy(t *testing.T) {
	p := NewProcessor()
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}

	withStubbedFns(t, func() {
		cloneGitFn = func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
			return nil, nil, nil
		}
		isValidForBoneCloneFn = func(repo *gogit.Repository, config Config) (bool, error) { return false, nil }
		copyCalled := false
		copyFilesFn = func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error {
			copyCalled = true
			return nil
		}

		if err := p.Process(repo, pp, cfg); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if copyCalled {
			t.Fatalf("expected CopyFiles not to be called when repo is not valid")
		}
	})
}

func TestProcessor_Process_Valid_CopySuccess(t *testing.T) {
	p := NewProcessor()
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}

	withStubbedFns(t, func() {
		cloneGitFn = func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
			return nil, nil, nil
		}
		isValidForBoneCloneFn = func(repo *gogit.Repository, config Config) (bool, error) { return true, nil }
		copyCalled := false
  copyFilesFn = func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error {
			copyCalled = true
			return nil
		}

		if err := p.Process(repo, pp, cfg); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !copyCalled {
			t.Fatalf("expected CopyFiles to be called when repo is valid")
		}
	})
}

func TestProcessor_Process_CopyError(t *testing.T) {
	p := NewProcessor()
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}

	withStubbedFns(t, func() {
		cloneGitFn = func(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
			return nil, nil, nil
		}
		isValidForBoneCloneFn = func(repo *gogit.Repository, config Config) (bool, error) { return true, nil }
		copyFilesFn = func(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig) error {
			return errors.New("cperr")
		}

		err := p.Process(repo, pp, cfg)
		if err == nil || err.Error() != "copy: cperr" {
			t.Fatalf("expected copy error wrapping, got: %v", err)
		}
	})
}
