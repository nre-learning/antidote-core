package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "antidote-stats"
	app.Usage = "A CLI tool configure and start antidote-stats on influxDB"

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "start exporting data to influx TSDB",
			Action: func(c *cli.Context) error {
				var mockSyringeConfig = GetmockSyringeConfig()
				var curriculum = GetCurriculum(mockSyringeConfig)
				var mockLiveLessonState = GetMockLiveLessonState()

				aStats := AntidoteStats{
					Curriculum:      curriculum,
					InfluxURL:       mockSyringeConfig.InfluxURL,
					InfluxUsername:  mockSyringeConfig.InfluxUsername,
					InfluxPassword:  mockSyringeConfig.InfluxPassword,
					LiveLessonState: mockLiveLessonState,
					Tier:            mockSyringeConfig.Tier,
				}

				err := aStats.StartTSDBExport()

				return err
			},
		},
		{
			Name:  "config",
			Usage: "antidote-stats config <SUBCOMMAND>",
			Subcommands: []cli.Command{
				{
					Name:  "show",
					Usage: "show influxDB config",
					Action: func(c *cli.Context) error {
						var mockSyringeConfig = GetmockSyringeConfig()

						log.Info(fmt.Sprintf("InfluxDB URL:\t%s", mockSyringeConfig.InfluxURL))
						log.Info(fmt.Sprintf("InfluxDB Username:   %s", mockSyringeConfig.InfluxPassword))

						return nil
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
