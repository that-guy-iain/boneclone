package repository_providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"

	"go.iain.rocks/boneclone/app/domain"
)

// Small interfaces to allow testing without the full Azure DevOps clients.
type coreClient interface {
	GetProjects(context.Context, core.GetProjectsArgs) (*core.GetProjectsResponseValue, error)
}

type gitClient interface {
	GetRepositories(context.Context, git.GetRepositoriesArgs) (*[]git.GitRepository, error)
	CreatePullRequest(context.Context, git.CreatePullRequestArgs) (*git.GitPullRequest, error)
}

// Constructors are variables so tests can stub them.
var newCoreClient = func(ctx context.Context, conn *azuredevops.Connection) (coreClient, error) {
	return core.NewClient(ctx, conn)
}

var newGitClient = func(ctx context.Context, conn *azuredevops.Connection) (gitClient, error) {
	return git.NewClient(ctx, conn)
}

type AzureRepositoryProvider struct {
	connection *azuredevops.Connection
	ctx        context.Context
}

func (a AzureRepositoryProvider) CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody domain.PRBodyBuilder) (domain.PRInfo, error) {
	gc, err := newGitClient(ctx, a.connection)
	if err != nil {
		return domain.PRInfo{}, err
	}

	// Expect repo in the form "Project/Repository"
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return domain.PRInfo{}, fmt.Errorf("azure repo must be 'Project/Repository', got: %s", repo)
	}
	project := parts[0]
	repoName := parts[1]

	sourceRef := "refs/heads/" + headBranch
	targetRef := "refs/heads/" + baseBranch

	body := ""
	if buildBody != nil {
		body = buildBody(repo, baseBranch, headBranch, filesChanged, originalAuthor)
	}

	pr := &git.GitPullRequest{
		Title:         &title,
		Description:   &body,
		SourceRefName: &sourceRef,
		TargetRefName: &targetRef,
	}

	args := git.CreatePullRequestArgs{
		GitPullRequestToCreate: pr,
		Project:                &project,
		RepositoryId:           &repoName,
	}

	created, err := gc.CreatePullRequest(ctx, args)
	if err != nil {
		return domain.PRInfo{}, err
	}
	id := 0
	if created != nil && created.PullRequestId != nil {
		id = *created.PullRequestId
	}
	url := ""
	if created != nil && created.Url != nil {
		url = *created.Url
	}
	return domain.PRInfo{ID: id, URL: url}, nil
}

// AssignReviewers attempts to add reviewers to an Azure DevOps pull request.
// Reviewers are provided as unique names (e.g., email/UPN); unknown reviewers are ignored by the API.
// If the underlying git client doesn't expose the batch reviewers API (in tests), this is a no-op.
func (a AzureRepositoryProvider) AssignReviewers(ctx context.Context, repo string, pr domain.PRInfo, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}

	gc, err := newGitClient(ctx, a.connection)
	if err != nil {
		return err
	}

	// Expect repo in the form "Project/Repository" to resolve project and repo names.
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		// Keep behavior lenient: do not fail the whole flow; return nil to be consistent with best-effort semantics.
		return nil
	}
	project := parts[0]
	repoName := parts[1]

	// Define a narrow interface for the reviewers API to avoid widening our gitClient test seam.
	type prReviewerClient interface {
		CreatePullRequestReviewers(context.Context, git.CreatePullRequestReviewersArgs) (*[]webapi.IdentityRef, error)
	}

	rc, ok := gc.(prReviewerClient)
	if !ok {
		// In tests fakes may not implement reviewers API; treat as no-op.
		return nil
	}

	// Build identity refs using UniqueName to avoid a separate identity lookup.
	idents := make([]webapi.IdentityRef, 0, len(reviewers))
	for _, r := range reviewers {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		idents = append(idents, webapi.IdentityRef{UniqueName: &r})
	}
	if len(idents) == 0 {
		return nil
	}

	args := git.CreatePullRequestReviewersArgs{
		Reviewers:     &idents,
		Project:       &project,
		RepositoryId:  &repoName,
		PullRequestId: &pr.ID,
	}
	_, err = rc.CreatePullRequestReviewers(ctx, args)
	return err
}

func (a AzureRepositoryProvider) GetRepositories() (*[]domain.GitRepository, error) {
	var output []domain.GitRepository

	coreClient, err := newCoreClient(a.ctx, a.connection)
	getProjectsArgs := core.GetProjectsArgs{}
	if err != nil {
		return nil, err
	}

	gitClient, err := newGitClient(a.ctx, a.connection)
	if err != nil {
		return nil, err
	}

	projectsResponse, err := coreClient.GetProjects(a.ctx, getProjectsArgs)
	if err != nil {
		return nil, err
	}

	trueValue := true
	for _, project := range projectsResponse.Value {
		getReposArgs := git.GetRepositoriesArgs{
			Project:        project.Name,
			IncludeHidden:  &trueValue,
			IncludeAllUrls: &trueValue,
			IncludeLinks:   &trueValue,
		}

		repositories, err := gitClient.GetRepositories(a.ctx, getReposArgs)
		if err != nil {
			return nil, err
		}

		if repositories != nil {
			for _, repo := range *repositories {
				output = append(output, domain.GitRepository{
					Name: *project.Name,
					Url:  *repo.RemoteUrl,
				})
			}
		}
	}

	return &output, nil
}

func NewAzureRepositoryProvider(token, org string) (domain.GitRepositoryProvider, error) {
	connection := azuredevops.NewPatConnection(org, token)
	ctx := context.Background()

	return &AzureRepositoryProvider{connection, ctx}, nil
}
