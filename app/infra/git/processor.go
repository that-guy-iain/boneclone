package git

import (
    "fmt"

    "go.iain.rocks/boneclone/app/domain"
)

// Processor implements domain.RepoProcessor using the functions in this package.
type Processor struct{}

func NewProcessor() *Processor { return &Processor{} }

func (p *Processor) Process(repo domain.GitRepository, pp domain.ProviderConfig, config domain.Config) error {
    fmt.Printf("repo: %s\n", repo.Url)

    gitRepo, fs, err := CloneGit(repo, pp)
    if err != nil {
        return fmt.Errorf("clone: %w", err)
    }

    valid, err := IsValidForBoneClone(gitRepo, config)
    if err != nil {
        return fmt.Errorf("validate: %w", err)
    }

    if valid {
        if err := CopyFiles(gitRepo, fs, config.Files, pp); err != nil {
            return fmt.Errorf("copy: %w", err)
        }
    }

    return nil
}
