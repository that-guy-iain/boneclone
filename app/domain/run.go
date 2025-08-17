package domain

import (
	"context"
	"fmt"
	"sync"
)

type ProviderFactory func(ProviderConfig) (GitRepositoryProvider, error)

type RepoProcessor interface {
	Process(repo GitRepository, provider ProviderConfig, config Config) error
}

func Run(_ context.Context, config Config, newProvider ProviderFactory, processor RepoProcessor) error {
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
