package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"superspreader/app/domain"

	"github.com/urfave/cli/v3"

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
				Value:   ".superspreader.yaml",
				Usage:   "The config file to be used",
				Aliases: []string{"c"},
			},
		},
		Name:  "run",
		Usage: "Run superspreader",
		Action: func(cxt context.Context, c *cli.Command) error {
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
				fmt.Printf("%v\n", pp.Provider)
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
