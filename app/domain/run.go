package domain

import (
    "context"
    "fmt"
    "sync"
)

// ProviderFactory constructs a GitRepositoryProvider for a given provider config.
type ProviderFactory func(ProviderConfig) (GitRepositoryProvider, error)

// RepoProcessor encapsulates the per-repository workflow (clone, validate, copy, push).
// Implementations should be side-effecting and idempotent where possible.
type RepoProcessor interface {
    Process(repo GitRepository, provider ProviderConfig, config Config) error
}

// Run executes the core BoneClone workflow for the provided configuration.
// It discovers repositories from configured providers and delegates per-repo
// work to the provided RepoProcessor. Errors are logged per-repo and processing
// continues; the function returns after all goroutines complete.
func Run(ctx context.Context, config Config, newProvider ProviderFactory, processor RepoProcessor) error { // ctx reserved for future cancellation/logging
    var wg sync.WaitGroup

    for _, pp := range config.Providers {
        provider, err := newProvider(pp)
        if err != nil {
            fmt.Printf("error creating provider %s: %v\n", pp.Provider, err)
            continue
        }

        repositories, err := provider.GetRepositories()
        if err != nil {
            fmt.Printf("error listing repositories for provider %s: %v\n", pp.Provider, err)
            continue
        }

        for _, repo := range *repositories {
            wg.Add(1)
            go func(repo GitRepository, pp ProviderConfig) {
                defer wg.Done()
                if err := processor.Process(repo, pp, config); err != nil {
                    fmt.Printf("error processing repo %s: %v\n", repo.Url, err)
                }
            }(repo, pp)
        }
    }

    wg.Wait()
    return nil
}
