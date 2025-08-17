package repository_providers

import (
	"context"

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

func NewAzureRepositoryProvider(token string, org string) (domain.GitRepositoryProvider, error) {
	connection := azuredevops.NewPatConnection(org, token)
	ctx := context.Background()

	return &AzureRepositoryProvider{connection, ctx}, nil
}
