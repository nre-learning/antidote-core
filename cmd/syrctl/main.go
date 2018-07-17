package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	cli "github.com/codegangsta/cli"
	grpc "google.golang.org/grpc"
	// log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func main() {

	type APIExpClient struct {
		Conf map[string]string
	}
	var client APIExpClient

	app := cli.NewApp()
	app.Name = "cyrctl"
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

		// TODO(mierdin) need to document usage of c.Args().First()
		{
			Name:    "lab",
			Aliases: []string{"labs"},
			Usage:   "Work with Syringe Labs",
			Subcommands: []cli.Command{
				{
					Name:  "get",
					Usage: "Retrieve a single lab",
					Action: func(c *cli.Context) {
						var (
							serverAddr = flag.String("server_addr", "127.0.0.1:50099", "The server address in the format of host:port")
						)

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLabsClient(conn)

						labUUID, _ := strconv.Atoi(
							c.Args().First(),
						)

						labdetails, err := client.GetLab(context.Background(), &pb.LabUUID{Id: int32(labUUID)})
						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(labdetails)

					},
				},
			},
		},
	}

	app.Run(os.Args)
}
