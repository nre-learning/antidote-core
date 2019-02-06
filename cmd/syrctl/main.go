package main

import (
	"context"
	"fmt"
	"os"

	cli "github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/nre-learning/syringe/api/exp"
	"github.com/nre-learning/syringe/config"
	grpc "google.golang.org/grpc"

	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func main() {

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
		{
			Name:    "whitelist",
			Aliases: []string{"wl"},
			Usage:   "syrctl whitelist <subcommand>",
			Subcommands: []cli.Command{
				{
					Name:  "show",
					Usage: "Show whitelisted sessions",
					Action: func(c *cli.Context) {

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						whitelist, err := client.GetGCWhitelist(context.Background(), &empty.Empty{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						sessions := whitelist.Sessions

						if len(sessions) == 0 {
							fmt.Println("No exempt sessions found.")
							os.Exit(0)
						}

						fmt.Println("EXEMPT SESSIONS")

						for i := range sessions {
							fmt.Println(sessions[i].Id)
						}
					},
				},
				{
					Name:  "add",
					Usage: "Add session to GC whitelist",
					Action: func(c *cli.Context) {

						sid := c.Args().First()
						if sid == "" {
							fmt.Println("Please provide session ID to add to whitelist")
							os.Exit(1)
						}

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						client.AddSessiontoGCWhitelist(context.Background(), &pb.Session{Id: sid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						fmt.Printf("%s added to whitelist\n", sid)
					},
				},
				{
					Name:  "remove",
					Usage: "Remove session from GC whitelist",
					Action: func(c *cli.Context) {

						sid := c.Args().First()
						if sid == "" {
							fmt.Println("Please provide session ID to remove from whitelist")
							os.Exit(1)
						}

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						client.RemoveSessionFromGCWhitelist(context.Background(), &pb.Session{Id: sid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						fmt.Printf("%s removed from whitelist\n", sid)
					},
				},
			},
		},
		{
			Name:    "livelesson",
			Aliases: []string{"ll"},
			Usage:   "syrctl livelesson <subcommand>",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List livelessons",
					Action: func(c *cli.Context) {

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
					},
				},
				{
					Name:  "kill",
					Usage: "Kill a livelesson",
					Action: func(c *cli.Context) {

						sid := c.Args().First()
						if sid == "" {
							fmt.Println("Please provide livelesson ID to kill")
							os.Exit(1)
						}

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						client.RemoveSessionFromGCWhitelist(context.Background(), &pb.Session{Id: sid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						fmt.Printf("Livelesson %s killed.\n", sid)
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
