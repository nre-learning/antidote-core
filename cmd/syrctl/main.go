package main

import (
	"os"

	cli "github.com/codegangsta/cli"
	"github.com/fatih/color"
	api "github.com/nre-learning/syringe/api/exp"
	"github.com/nre-learning/syringe/config"
)

func main() {

	type APIExpClient struct {
		Conf map[string]string
	}
	var client APIExpClient

	app := cli.NewApp()
	app.Name = "syrctl"
	app.Version = "v0.1.0"
	app.Usage = "Scheduling for the Antidote project and NRE Labs"

	var host, port string

	// global level flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "H, host",
			Usage:       "syringed hostname",
			Value:       "127.0.0.1",
			Destination: &host,
		},
		cli.StringFlag{
			Name:        "P, port",
			Usage:       "syringed port",
			Value:       "50099",
			Destination: &port,
		},
	}

	// TODO(mierdin): This MAY not work. These vars may not execute until after app.Run
	client.Conf = map[string]string{
		"host": host,
		"port": port,
	}

	app.Commands = []cli.Command{
		{
			Name:    "validate",
			Aliases: []string{"validate"},
			Usage:   "syrctl validate <LESSON DIRECTORY>",
			Action: func(c *cli.Context) {

				_, err := api.ImportLessonDefs(&config.SyringeConfig{Tier: "local"}, c.Args().First())
				if err != nil {
					color.Red("Some lessons failed to validate.")
					os.Exit(1)
				} else {
					color.Green("All detected lesson files imported successfully.")
					os.Exit(0)
				}
			},
		},
	}

	app.Run(os.Args)
}
