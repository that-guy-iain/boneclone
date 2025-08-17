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

// Small interfaces for merge requests and users to enable testing without real client.
// Only the methods used are included.
type gitlabMergeRequestService interface {
	CreateMergeRequest(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
	UpdateMergeRequest(pid interface{}, mergeRequestIID int, opt *gitlab.UpdateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
}

type gitlabUserLister interface {
	ListUsers(opt *gitlab.ListUsersOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.User, *gitlab.Response, error)
}

type GitlabRepositoryProvider struct {
	groups gitlabGroupProjectLister
	mrs    gitlabMergeRequestService
	users  gitlabUserLister
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
	mr, _, err := g.mrs.CreateMergeRequest(pid, opt, gitlab.WithContext(ctx))
	if err != nil {
		return domain.PRInfo{}, err
	}
	return domain.PRInfo{ID: mr.IID, URL: mr.WebURL}, nil
}

// AssignReviewers sets reviewers on an existing merge request by resolving usernames to user IDs.
// Unknown reviewers are ignored; if none can be resolved, this is a no-op.
func (g GitlabRepositoryProvider) AssignReviewers(ctx context.Context, repo string, pr domain.PRInfo, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	var ids []int
	seen := map[int]struct{}{}
	for _, uname := range reviewers {
		if uname == "" {
			continue
		}
		opt := &gitlab.ListUsersOptions{Username: &uname}
		users, _, err := g.users.ListUsers(opt, gitlab.WithContext(ctx))
		if err != nil {
			return err
		}
		for _, u := range users {
			if u != nil && u.Username == uname {
				if _, ok := seen[u.ID]; !ok {
					ids = append(ids, u.ID)
					seen[u.ID] = struct{}{}
				}
				break
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	pid := fmt.Sprintf("%s/%s", g.org, repo)
	opt := &gitlab.UpdateMergeRequestOptions{ReviewerIDs: &ids}
	_, _, err := g.mrs.UpdateMergeRequest(pid, pr.ID, opt, gitlab.WithContext(ctx))
	return err
}

func NewGitlabRepositoryProvider(token, org string) (domain.GitRepositoryProvider, error) {
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}
	return &GitlabRepositoryProvider{groups: client.Groups, mrs: client.MergeRequests, users: client.Users, org: org}, nil
}
