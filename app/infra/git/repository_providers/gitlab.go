package repository_providers

import (
	"context"
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"go.iain.rocks/boneclone/app/domain"
)

// Small interface to allow testing without real GitLab client
// It matches the single method we use from the Groups service.
type gitlabGroupProjectLister interface {
	ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

// Small interface for creating merge requests to enable testing without real client
// Only the method used is included.
type gitlabMergeRequestCreator interface {
	CreateMergeRequest(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
}

type GitlabRepositoryProvider struct {
	groups gitlabGroupProjectLister
	mrs    gitlabMergeRequestCreator
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

	output := make([]domain.GitRepository, 0, len(allProjects))
	for _, project := range allProjects {
		output = append(output, domain.GitRepository{
			Name: project.Name,
			Url:  project.HTTPURLToRepo,
		})
	}

	return &output, nil
}

// CreatePullRequest creates a merge request on GitLab for the given repo (within the configured org/group).
// baseBranch is the target, headBranch is the source.
func (g GitlabRepositoryProvider) CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch string, filesChanged []string, originalAuthor string) error {
	title := "BoneClone update"
	var bodyBuilder strings.Builder
	bodyBuilder.WriteString("This is a BoneClone PR.\n\n")
	if originalAuthor != "" {
		bodyBuilder.WriteString(fmt.Sprintf("Original author: %s\n\n", originalAuthor))
	}
	if len(filesChanged) > 0 {
		bodyBuilder.WriteString("Files changed:\n")
		for _, f := range filesChanged {
			bodyBuilder.WriteString("- ")
			bodyBuilder.WriteString(f)
			bodyBuilder.WriteString("\n")
		}
	}
	body := bodyBuilder.String()

	opt := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		SourceBranch: &headBranch,
		TargetBranch: &baseBranch,
		Description:  &body,
	}

	pid := fmt.Sprintf("%s/%s", g.org, repo)
	_, _, err := g.mrs.CreateMergeRequest(pid, opt)
	return err
}

func NewGitlabRepositoryProvider(token, org string) (domain.GitRepositoryProvider, error) {
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}
	return &GitlabRepositoryProvider{groups: client.Groups, mrs: client.MergeRequests, org: org}, nil
}
