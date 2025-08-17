package repository_providers

import (
	"errors"
	"reflect"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"go.iain.rocks/boneclone/app/domain"
)

// fakeGroupsService implements gitlabGroupProjectLister for tests.
type fakeGroupsService struct {
	// pages is a slice of project slices to return on successive calls
	pages [][]*gitlab.Project
	// errs is a slice of errors to return on successive calls (nil for no error)
	errs []error
	call int
}

func (f *fakeGroupsService) ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	i := f.call
	f.call++
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	var projects []*gitlab.Project
	if i < len(f.pages) {
		projects = f.pages[i]
	} else {
		projects = []*gitlab.Project{}
	}
	// Determine next page: if another page exists, set NextPage to current+1
	next := 0
	if i+1 < len(f.pages) {
		next = opt.Page + 1
	}
	resp := &gitlab.Response{NextPage: next}
	return projects, resp, err
}

func TestGitlabProvider_GetRepositories_PaginationSuccess(t *testing.T) {
	// Arrange: two pages of projects
	p1 := &gitlab.Project{Name: "RepoA", HTTPURLToRepo: "https://gitlab.com/org/repoA.git"}
	p2 := &gitlab.Project{Name: "RepoB", HTTPURLToRepo: "https://gitlab.com/org/repoB.git"}
	p3 := &gitlab.Project{Name: "RepoC", HTTPURLToRepo: "https://gitlab.com/org/repoC.git"}

	fake := &fakeGroupsService{
		pages: [][]*gitlab.Project{{p1, p2}, {p3}},
		errs:  []error{nil, nil},
	}

	provider := &GitlabRepositoryProvider{groups: fake, org: "my-org"}

	// Act
	got, err := provider.GetRepositories()
	if err != nil {
		t.Fatalf("GetRepositories unexpected error: %v", err)
	}

	// Assert
	want := []domain.GitRepository{
		{Name: "RepoA", Url: "https://gitlab.com/org/repoA.git"},
		{Name: "RepoB", Url: "https://gitlab.com/org/repoB.git"},
		{Name: "RepoC", Url: "https://gitlab.com/org/repoC.git"},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("GetRepositories mismatch\nGot:  %#v\nWant: %#v", *got, want)
	}
}

func TestGitlabProvider_GetRepositories_ErrorPropagation(t *testing.T) {
	// First call errors, ensure it propagates
	fake := &fakeGroupsService{
		pages: [][]*gitlab.Project{{}},
		errs:  []error{errors.New("kaboom")},
	}
	provider := &GitlabRepositoryProvider{groups: fake, org: "org"}
	repos, err := provider.GetRepositories()
	if err == nil {
		t.Fatalf("expected error, got nil and repos=%v", repos)
	}
}

func TestGitlabProvider_GetRepositories_Empty(t *testing.T) {
	fake := &fakeGroupsService{pages: [][]*gitlab.Project{{}}, errs: []error{nil}}
	provider := &GitlabRepositoryProvider{groups: fake, org: "org"}
	got, err := provider.GetRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected empty slice pointer, got nil")
	}
	if len(*got) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(*got))
	}
}

func TestNewGitlabRepositoryProvider_Constructs(t *testing.T) {
	p, err := NewGitlabRepositoryProvider("token", "my-org")
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
