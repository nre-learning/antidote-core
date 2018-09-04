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

		// TODO(mierdin) need to document usage of c.Args().First()
		{
			Name:    "livelesson",
			Aliases: []string{"livelesson"},
			Usage:   "Work with Syringe livelesson",
			Subcommands: []cli.Command{
				{
					Name:  "get",
					Usage: "Retrieve a single livelab",
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
						client := pb.NewLiveLessonsServiceClient(conn)

						liveLabDetails, err := client.GetLiveLesson(context.Background(), &pb.LessonUUID{Id: c.Args().First()})
						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(liveLabDetails)
					},
				},
				{
					Name:  "request",
					Usage: "Request a new livelab",
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
						client := pb.NewLiveLessonsServiceClient(conn)

						lessonId, _ := strconv.Atoi(c.Args()[0])

						liveLabDetails, err := client.RequestLiveLesson(context.Background(), &pb.LessonParams{
							LessonId:  int32(lessonId),
							SessionId: c.Args()[1],
						})

						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(liveLabDetails.Id)

					},
				},
				{
					Name:  "delete",
					Usage: "Delete an existing livelab",
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
						client := pb.NewLiveLessonsServiceClient(conn)

						lessonId, _ := strconv.Atoi(c.Args()[0])

						_, err = client.DeleteLiveLesson(context.Background(), &pb.LessonParams{
							LessonId:  int32(lessonId),
							SessionId: c.Args()[1],
						})

						if err != nil {
							fmt.Println(err)
						}
						fmt.Println("Deleted.")

					},
				},
			},
		},
	}

	app.Run(os.Args)
}
