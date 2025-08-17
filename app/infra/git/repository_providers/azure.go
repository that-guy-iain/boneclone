package repository_providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"

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

// AssignReviewers is a best-effort no-op for Azure in this iteration.
// Future improvement: wire git.CreatePullRequestReviewer(s) once user identity resolution is added.
func (a AzureRepositoryProvider) AssignReviewers(ctx context.Context, repo string, pr domain.PRInfo, reviewers []string) error {
	return nil
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
