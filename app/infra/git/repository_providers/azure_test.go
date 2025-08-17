package repository_providers

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"

	"go.iain.rocks/boneclone/app/domain"
)

type fakeCoreClient struct {
	projects core.GetProjectsResponseValue
	err      error
}

func (f fakeCoreClient) GetProjects(ctx context.Context, args core.GetProjectsArgs) (*core.GetProjectsResponseValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	// Return a copy to avoid accidental mutation
	out := f.projects
	return &out, nil
}

type fakeGitClient struct {
	repos []git.GitRepository
	err   error
}

func (f fakeGitClient) GetRepositories(ctx context.Context, args git.GetRepositoriesArgs) (*[]git.GitRepository, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]git.GitRepository, len(f.repos))
	copy(out, f.repos)
	return &out, nil
}

func strPtr(s string) *string { return &s }

func TestAzureProvider_GetRepositories_Success(t *testing.T) {
	// Save and restore constructors
	origCore := newCoreClient
	origGit := newGitClient
	t.Cleanup(func() {
		newCoreClient = origCore
		newGitClient = origGit
	})

	// Arrange fake data: 2 projects with different repos
	p1 := core.TeamProjectReference{Name: strPtr("ProjectOne")}
	p2 := core.TeamProjectReference{Name: strPtr("ProjectTwo")}
	projects := core.GetProjectsResponseValue{
		Value: []core.TeamProjectReference{p1, p2},
	}

	r1 := git.GitRepository{RemoteUrl: strPtr("https://dev.azure.com/org/ProjectOne/_git/repo1")}
	r2 := git.GitRepository{RemoteUrl: strPtr("https://dev.azure.com/org/ProjectOne/_git/repo2")}
	r3 := git.GitRepository{RemoteUrl: strPtr("https://dev.azure.com/org/ProjectTwo/_git/repoA")}

	// Inject fakes
	newCoreClient = func(ctx context.Context, _ *azuredevops.Connection) (coreClient, error) { // connection not used in fake
		return fakeCoreClient{projects: projects}, nil
	}
	// First call for ProjectOne returns r1,r2; second call returns r3.
	call := 0
	newGitClient = func(ctx context.Context, _ *azuredevops.Connection) (gitClient, error) {
		call = 0 // ensure fresh counter per provider creation
		return fakeGitClient{}, nil
	}
	// We can't inject behavior via constructor easily, so instead wrap GetRepositories with a closure over call
	// Redefine newGitClient to return a stateful fake implementing the interface
	newGitClient = func(ctx context.Context, _ *azuredevops.Connection) (gitClient, error) {
		return &statefulGitFake{calls: &call, slices: [][]git.GitRepository{{r1, r2}, {r3}}}, nil
	}

	// Act
	provider := &AzureRepositoryProvider{connection: nil, ctx: context.Background()}
	got, err := provider.GetRepositories()
	if err != nil {
		t.Fatalf("GetRepositories unexpected error: %v", err)
	}

	// Assert
	want := []domain.GitRepository{
		{Name: "ProjectOne", Url: *r1.RemoteUrl},
		{Name: "ProjectOne", Url: *r2.RemoteUrl},
		{Name: "ProjectTwo", Url: *r3.RemoteUrl},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("GetRepositories mismatch\nGot:  %#v\nWant: %#v", *got, want)
	}
}

type statefulGitFake struct {
	calls  *int
	slices [][]git.GitRepository
}

func (s *statefulGitFake) GetRepositories(ctx context.Context, args git.GetRepositoriesArgs) (*[]git.GitRepository, error) {
	i := *s.calls
	if i >= len(s.slices) {
		empty := []git.GitRepository{}
		return &empty, nil
	}
	out := s.slices[i]
	*s.calls = i + 1
	return &out, nil
}

func TestAzureProvider_GetRepositories_CoreError(t *testing.T) {
	origCore := newCoreClient
	origGit := newGitClient
	t.Cleanup(func() { newCoreClient = origCore; newGitClient = origGit })

	newCoreClient = func(ctx context.Context, _ *azuredevops.Connection) (coreClient, error) {
		return fakeCoreClient{err: errors.New("boom")}, nil
	}
	newGitClient = func(ctx context.Context, _ *azuredevops.Connection) (gitClient, error) {
		return fakeGitClient{}, nil
	}

	provider := &AzureRepositoryProvider{connection: nil, ctx: context.Background()}
	repos, err := provider.GetRepositories()
	if err == nil {
		t.Fatalf("expected error, got nil and repos=%v", repos)
	}
}

func TestNewAzureRepositoryProvider_Constructs(t *testing.T) {
	p, err := NewAzureRepositoryProvider("token", "https://dev.azure.com/org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatalf("expected provider, got nil")
	}
	if _, ok := p.(domain.GitRepositoryProvider); !ok {
		t.Fatalf("provider does not implement domain.GitRepositoryProvider")
	}
}
