package main

import (
	"context"
	"fmt"
	"github.com/that-guy-iain/boneclone/app/domain"
	"github.com/that-guy-iain/boneclone/app/infra/git"
	"github.com/that-guy-iain/boneclone/app/infra/git/repository_providers"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"sync"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var conf = koanf.Conf{
	Delim:       ".",
	StrictMerge: true,
}
var k = koanf.NewWithConf(conf)

func main() {

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Value:   ".github.com/that-guy-iain/boneclone.yaml",
				Usage:   "The config file to be used",
				Aliases: []string{"c"},
			},
		},
		EnableShellCompletion: true,
		Name:                  "run",
		Usage:                 "Run BoneClone",
		Action: func(cxt context.Context, c *cli.Command) error {
			var wg sync.WaitGroup
			configFile := c.String("config")

			if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
				log.Fatalf("error loading config: %v", err)
			}

			var config domain.Config
			err := k.Unmarshal("", &config)

			if err != nil {
				log.Fatalf("error unmarshalling config: %v", err)
			}
			for _, pp := range config.Providers {

				provider, _ := repository_providers.NewProvider(pp)
				repositories, _ := provider.GetRepositories()

				for _, repo := range repositories {
					wg.Add(1)
					go func(repo domain.GitRepository) {
						defer wg.Done()

						fmt.Printf("repo: %s\n", repo.Url)
						gitRepo, fs, err := git.CloneGit(repo, pp)

						if err != nil {
							fmt.Printf("error cloning repo: %v\n", err)
							return
						}

						valid, err := git.IsValidForBoneClone(gitRepo, config)

						if err != nil {
							fmt.Printf("error checking repo: %v\n", err)

							return
						}
						if valid {

							files := git.CopyFiles(gitRepo, fs, config.Files, pp)
							if files != nil {
								fmt.Printf("files: %v\n", files)
							}
						}
					}(repo)
				}

			}
			wg.Wait()
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
