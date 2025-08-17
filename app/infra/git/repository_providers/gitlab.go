package repository_providers

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.iain.rocks/boneclone/app/domain"
)

// Small interface to allow testing without real GitLab client
// It matches the single method we use from the Groups service.
type gitlabGroupProjectLister interface {
	ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

type GitlabRepositoryProvider struct {
	groups gitlabGroupProjectLister
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
		projects, resp, err := g.groups.ListGroupProjects(g.org, opts)
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
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}
	return &GitlabRepositoryProvider{groups: client.Groups, org: org}, nil
}
