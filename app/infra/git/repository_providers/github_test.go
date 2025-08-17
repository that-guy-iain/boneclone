package repository_providers

import (
    "net/http"
    "net/http/httptest"
    "net/url"
    "reflect"
    "testing"

    github "github.com/google/go-github/v72/github"
    "go.iain.rocks/boneclone/app/domain"
)

func TestGithubProvider_GetRepositories_Success(t *testing.T) {
    // Arrange a fake GitHub API server
    org := "my-org"
    reposJSON := `[
        {"name": "repo-one", "clone_url": "https://github.com/my-org/repo-one.git"},
        {"name": "repo-two", "clone_url": "https://github.com/my-org/repo-two.git"}
    ]`

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/orgs/"+org+"/repos" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(reposJSON))
    }))
    defer srv.Close()

    // Create a go-github client pointed at the fake server
    httpClient := srv.Client()
    client := github.NewClient(httpClient)
    base, err := url.Parse(srv.URL + "/")
    if err != nil {
        t.Fatalf("parse url: %v", err)
    }
    client.BaseURL = base

    provider := &GithubRepositoryProvider{github: client, orgName: org}

    // Act
    got, err := provider.GetRepositories()
    if err != nil {
        t.Fatalf("GetRepositories unexpected error: %v", err)
    }

    // Assert
    want := []domain.GitRepository{
        {Name: "repo-one", Url: "https://github.com/my-org/repo-one.git"},
        {Name: "repo-two", Url: "https://github.com/my-org/repo-two.git"},
    }
    if !reflect.DeepEqual(*got, want) {
        t.Fatalf("GetRepositories mismatch\nGot:  %#v\nWant: %#v", *got, want)
    }
}

func TestGithubProvider_GetRepositories_APIError(t *testing.T) {
    org := "acme"
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/orgs/"+org+"/repos" {
            http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
            return
        }
        http.NotFound(w, r)
    }))
    defer srv.Close()

    httpClient := srv.Client()
    client := github.NewClient(httpClient)
    base, _ := url.Parse(srv.URL + "/")
    client.BaseURL = base

    provider := &GithubRepositoryProvider{github: client, orgName: org}

    repos, err := provider.GetRepositories()
    if err == nil {
        t.Fatalf("expected error, got nil and repos=%v", repos)
    }
}

func TestNewGithubRepositoryProvider_Constructs(t *testing.T) {
    p, err := NewGithubRepositoryProvider("token", "any-org")
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
