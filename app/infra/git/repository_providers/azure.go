package repository_providers

import (
	"context"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/that-guy-iain/boneclone/app/domain"
)

type AzureRepositoryProvider struct {
	connection *azuredevops.Connection
	ctx        context.Context
}

func (a AzureRepositoryProvider) GetRepositories() ([]domain.GitRepository, error) {
	var output []domain.GitRepository

	coreClient, err := core.NewClient(a.ctx, a.connection)
	getProjectsArgs := core.GetProjectsArgs{}
	if err != nil {
		return output, err
	}

	gitClient, err := git.NewClient(a.ctx, a.connection)
	if err != nil {
		return output, err
	}

	projectsResponse, err := coreClient.GetProjects(a.ctx, getProjectsArgs)
	if err != nil {
		return output, err
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
			return output, err
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

	return output, nil
}

func NewAzureRepositoryProvider(token string, org string) (domain.GitRepositoryProvider, error) {
	connection := azuredevops.NewPatConnection(org, token)
	ctx := context.Background()

	return &AzureRepositoryProvider{connection, ctx}, nil
}
