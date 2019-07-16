package main

import (
	"context"
	"fmt"
	"os"

	cli "github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nre-learning/syringe/config"
	grpc "google.golang.org/grpc"

	api "github.com/nre-learning/syringe/api/exp"
	pb "github.com/nre-learning/syringe/api/exp/generated"
)

func main() {

	app := cli.NewApp()
	app.Name = "syrctl"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "Command-line tool to interact with Syringe, the scheduler for Antidote"

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
			Usage:   "syrctl validate <CURRICULUM DIRECTORY>",
			Action: func(c *cli.Context) {

				_, err := api.ImportCurriculum(&config.SyringeConfig{
					Tier:          "local",
					CurriculumDir: c.Args().First(),
				})
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				} else {
					color.Green("All detected curriculum resources imported successfully.")
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

						_, err = client.RemoveSessionFromGCWhitelist(context.Background(), &pb.Session{Id: sid})
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
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						liveLessons, err := client.ListLiveLessons(context.Background(), &empty.Empty{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						for i := range liveLessons.Items {
							fmt.Println(liveLessons.Items[i].LessonUUID)
						}
					},
				},
				{
					Name:  "get",
					Usage: "get a Livelesson",
					Action: func(c *cli.Context) {

						uuid := c.Args().First()
						if uuid == "" {
							fmt.Println("Please provide livelesson ID to get")
							os.Exit(1)
						}

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						ll, err := client.GetLiveLesson(context.Background(), &pb.LessonUUID{Id: uuid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						jpbm := jsonpb.Marshaler{}
						fmt.Println(jpbm.MarshalToString(ll))
					},
				},
				{
					Name:  "kill",
					Usage: "Kill a livelesson",
					Action: func(c *cli.Context) {

						uuid := c.Args().First()
						if uuid == "" {
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

						_, err = client.KillLiveLesson(context.Background(), &pb.LessonUUID{Id: uuid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						fmt.Printf("Livelesson %s killed.\n", uuid)
					},
				},
			},
		},

		{
			Name:    "kubelab",
			Aliases: []string{"kl"},
			Usage:   "syrctl kubelab <subcommand>",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List kubelabs",
					Action: func(c *cli.Context) {

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewKubeLabServiceClient(conn)

						kubeLabs, err := client.ListKubeLabs(context.Background(), &empty.Empty{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						for uuid := range kubeLabs.Items {
							fmt.Println(uuid)
						}

					},
				},
				{
					Name:  "get",
					Usage: "get a Kubelab",
					Action: func(c *cli.Context) {

						uuid := c.Args().First()
						if uuid == "" {
							fmt.Println("Please provide kubelab ID to get")
							os.Exit(1)
						}

						// TODO(mierdin): Add security options
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
						}
						defer conn.Close()
						client := pb.NewKubeLabServiceClient(conn)

						kubeLab, err := client.GetKubeLab(context.Background(), &pb.KubeLabUuid{Id: uuid})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						jpbm := jsonpb.Marshaler{}
						fmt.Println(jpbm.MarshalToString(kubeLab))
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
