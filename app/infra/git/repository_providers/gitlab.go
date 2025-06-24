package repository_providers

import (
	"github.com/that-guy-iain/boneclone/app/domain"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GitlabRepositoryProvider struct {
	gitlab *gitlab.Client
	org    string
}

func (g GitlabRepositoryProvider) GetRepositories() (*[]domain.GitRepository, error) {

	trueValue := true
	opts := &gitlab.ListGroupProjectsOptions{
		IncludeSubGroups: &trueValue,
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	var output []domain.GitRepository
	var allProjects []*gitlab.Project
	for {

		projects, resp, err := g.gitlab.Groups.ListGroupProjects(g.org, opts)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break // No more pages
		}
		opts.Page = resp.NextPage
	}

	for _, project := range allProjects {

		output = append(output, domain.GitRepository{
			Name: project.Name,
			Url:  project.HTTPURLToRepo,
		})
	}

	return &output, nil
}

func NewGitlabRepositoryProvider(token string, org string) (domain.GitRepositoryProvider, error) {
	gitlab, err := gitlab.NewClient(token)

	if err != nil {
		return nil, err
	}

	return &GitlabRepositoryProvider{gitlab: gitlab, org: org}, nil
}
