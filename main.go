package main

import (
	"log"
	"os"

	"github.com/nicksellen/pictures/gather"
	"github.com/nicksellen/pictures/index"
	"github.com/nicksellen/pictures/search"
	"github.com/nicksellen/pictures/server"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "pictures"
	app.Usage = "manage pictures!"

	app.Commands = []cli.Command{
		{
			Name:      "gather",
			Aliases:   []string{"a"},
			Usage:     "gather information I might want",
			ArgsUsage: "b2://bucketname/path/with/prefix",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "b2-account-id",
					Usage:  "b2 account id",
					EnvVar: "B2_ACCOUNT_ID",
				},
				cli.StringFlag{
					Name:   "b2-account-key",
					Usage:  "b2 account key",
					EnvVar: "B2_ACCOUNT_KEY",
				},
			},
			Action: func(c *cli.Context) error {

				b2id := c.String("b2-account-id")
				b2key := c.String("b2-account-key")

				if b2id == "" || b2key == "" {
					log.Fatal("must have b2 args or env vars")
				}

				path := c.Args().First()

				if path == "" {
					log.Fatal("must pass b2 path as arg")
				}

				return gather.Run(b2id, b2key, path)
			},
		},
		{
			Name:  "index",
			Usage: "fiddle about with indexing",
			Action: func(c *cli.Context) error {
				return index.Run()
			},
		},
		{
			Name:  "server",
			Usage: "run web server",
			Action: func(c *cli.Context) error {
				return server.Run()
			},
		},
		{
			Name:  "search",
			Usage: "search the index",
			Action: func(c *cli.Context) error {
				return search.Run()
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
