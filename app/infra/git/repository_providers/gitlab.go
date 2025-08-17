package repository_providers

import (
	"context"
	"fmt"

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
// The merge request body is produced by the provided buildBody function.
func (g GitlabRepositoryProvider) CreatePullRequest(ctx context.Context, repo, baseBranch, headBranch, title string, filesChanged []string, originalAuthor string, buildBody domain.PRBodyBuilder) (domain.PRInfo, error) {
	body := ""
	if buildBody != nil {
		body = buildBody(repo, baseBranch, headBranch, filesChanged, originalAuthor)
	}

	opt := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		SourceBranch: &headBranch,
		TargetBranch: &baseBranch,
		Description:  &body,
	}

	pid := fmt.Sprintf("%s/%s", g.org, repo)
	mr, _, err := g.mrs.CreateMergeRequest(pid, opt)
	if err != nil {
		return domain.PRInfo{}, err
	}
	return domain.PRInfo{ID: mr.IID, URL: mr.WebURL}, nil
}

// AssignReviewers is currently a best-effort no-op due to lack of username->ID mapping in this package.
// The processor treats assignment errors as non-fatal, so this returns nil.
func (g GitlabRepositoryProvider) AssignReviewers(ctx context.Context, repo string, pr domain.PRInfo, reviewers []string) error {
	return nil
}

func NewGitlabRepositoryProvider(token, org string) (domain.GitRepositoryProvider, error) {
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}
	return &GitlabRepositoryProvider{groups: client.Groups, mrs: client.MergeRequests, org: org}, nil
}
