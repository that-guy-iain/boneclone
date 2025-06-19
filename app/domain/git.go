package domain

type GitRepositoryProvider interface {
	GetRepositories() []GitRepository
}

type GitRepository struct {
	Name string
	Url  string
}
