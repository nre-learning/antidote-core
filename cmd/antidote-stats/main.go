package main

import (
	"os"
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
				var mockSyringeConfig = GetmockSyringeConfig(false)
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
	}

	app.Run(os.Args)
}
