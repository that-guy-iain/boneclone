package main

import (
	"context"
	"log"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"

	"go.iain.rocks/boneclone/app/domain"
	"go.iain.rocks/boneclone/app/infra/git"
	"go.iain.rocks/boneclone/app/infra/git/repository_providers"
)

var conf = koanf.Conf{
	Delim:       ".",
	StrictMerge: true,
}
var k = koanf.NewWithConf(conf)

func runWithArgs(args []string) error {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Value:   ".boneclone.yaml",
				Usage:   "The config file to be used",
				Aliases: []string{"c"},
			},
		},
		EnableShellCompletion: true,
		Name:                  "run",
		Usage:                 "Run BoneClone",
		Action: func(cxt context.Context, c *cli.Command) error {
			configFile := c.String("config")

			if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
				log.Fatalf("error loading config: %v", err)
			}

			// Expand environment variables in config values before unmarshalling
			if err := expandEnvValues(k); err != nil {
				log.Fatalf("error expanding env in config: %v", err)
			}

			// Set defaults for missing config values
			if !k.Exists("git.pullRequest") {
				if err := k.Set("git.pullRequest", false); err != nil {
					log.Fatalf("error setting default git.pullRequest: %v", err)
				}
			}
			if !k.Exists("git.targetBranch") {
				if err := k.Set("git.targetBranch", "main"); err != nil {
					log.Fatalf("error setting default git.targetBranch: %v", err)
				}
			}

			var config domain.Config
			if err := k.Unmarshal("", &config); err != nil {
				log.Fatalf("error unmarshalling config: %v", err)
			}

			// Configure skeleton name for PR messages (used in PR body)
			domain.SetSkeletonName(config.Name)

			ops := git.NewOperations()
			processor := domain.NewProcessorForConfig(config, ops, repository_providers.NewProvider)
			return domain.Run(cxt, config, repository_providers.NewProvider, processor)
		},
	}

	return cmd.Run(context.Background(), args)
}

func main() {
	if err := runWithArgs(os.Args); err != nil {
		log.Fatal(err)
	}
}

func expandEnvValues(k *koanf.Koanf) error {
	for _, key := range k.Keys() {
		val := k.Get(key)

		if strVal, ok := val.(string); ok {
			expandedVal := os.ExpandEnv(strVal)
			if expandedVal != strVal {
				if err := k.Set(key, expandedVal); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
