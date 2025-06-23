package domain

type GitRepositoryProvider interface {
	GetRepositories() ([]GitRepository, error)
}

type GitRepository struct {
	Name string
	Url  string
}
