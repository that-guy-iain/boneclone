package domain

import (
	"errors"
	"testing"

	billy "github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v6"
)

type fakeOps struct {
	cloneErr   error
	valid      bool
	validErr   error
	copyErr    error
	copyCalled bool
}

func (f *fakeOps) CloneGit(repo GitRepository, config ProviderConfig) (*gogit.Repository, billy.Filesystem, error) {
	return nil, nil, f.cloneErr
}
func (f *fakeOps) IsValidForBoneClone(repo *gogit.Repository, config Config) (bool, RemoteConfig, error) {
	return f.valid, RemoteConfig{}, f.validErr
}
func (f *fakeOps) CopyFiles(repo *gogit.Repository, fs billy.Filesystem, cfg Config, pp ProviderConfig, targetBranch string) error {
	f.copyCalled = true
	return f.copyErr
}

func TestProcessor_Process_CloneError(t *testing.T) {
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}
	ops := &fakeOps{cloneErr: errors.New("boom")}
	p := NewProcessor(ops)

	err := p.Process(repo, pp, cfg)
	if err == nil || err.Error() != "clone: boom" {
		t.Fatalf("expected clone error wrapping, got: %v", err)
	}
}

func TestProcessor_Process_ValidateError(t *testing.T) {
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}
	ops := &fakeOps{validErr: errors.New("valerr")}
	p := NewProcessor(ops)

	err := p.Process(repo, pp, cfg)
	if err == nil || err.Error() != "validate: valerr" {
		t.Fatalf("expected validate error wrapping, got: %v", err)
	}
}

func TestProcessor_Process_NotValid_SkipsCopy(t *testing.T) {
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}
	ops := &fakeOps{valid: false}
	p := NewProcessor(ops)

	if err := p.Process(repo, pp, cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ops.copyCalled {
		t.Fatalf("expected CopyFiles not to be called when repo is not valid")
	}
}

func TestProcessor_Process_Valid_CopySuccess(t *testing.T) {
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}
	ops := &fakeOps{valid: true}
	p := NewProcessor(ops)

	if err := p.Process(repo, pp, cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ops.copyCalled {
		t.Fatalf("expected CopyFiles to be called when repo is valid")
	}
}

func TestProcessor_Process_CopyError(t *testing.T) {
	repo := GitRepository{Url: "https://example.com/repo.git"}
	pp := ProviderConfig{}
	cfg := Config{}
	ops := &fakeOps{valid: true, copyErr: errors.New("cperr")}
	p := NewProcessor(ops)

	err := p.Process(repo, pp, cfg)
	if err == nil || err.Error() != "copy: cperr" {
		t.Fatalf("expected copy error wrapping, got: %v", err)
	}
}
