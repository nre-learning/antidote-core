package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"
	pb "github.com/nre-learning/antidote-core/api/exp/generated"

	"github.com/golang/protobuf/ptypes/empty"
	cli "github.com/urfave/cli"
	"google.golang.org/grpc"
)

func main() {

	app := cli.NewApp()
	app.Name = "antictl"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "Admin/debug tool for the Antidote platform"
	var host, port string

	// Ensure the server/client versions are identical
	app.Before = func(c *cli.Context) error {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer conn.Close()
		client := pb.NewAntidoteInfoServiceClient(conn)
		info, err := client.GetAntidoteInfo(context.Background(), &empty.Empty{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if info.GetBuildVersion() != buildInfo["buildVersion"] {
			color.Red("ERROR - server/client version mismatch")
			fmt.Printf("Server version is %s, client is %s\n", info.GetBuildVersion(), buildInfo["buildVersion"])
			os.Exit(1)
		}
		return nil
	}

	// global level flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "H, host",
			Usage:       "antidoted hostname",
			Value:       "127.0.0.1",
			Destination: &host,
		},
		cli.StringFlag{
			Name:        "P, port",
			Usage:       "antidoted grpc port",
			Value:       "50099",
			Destination: &port,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "lesson",
			Aliases: []string{},
			Usage:   "Work with Lesson resources",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "Retrieve all lessons and display in a table",
					Action: func(c *cli.Context) {
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						defer conn.Close()
						client := pb.NewLessonServiceClient(conn)

						lessons, err := client.ListLessons(context.Background(), &pb.LessonFilter{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						lJSON, _ := json.Marshal(lessons)
						fmt.Println(string(lJSON))

					},
				},
			},
		},
		{
			Name:    "livesession",
			Aliases: []string{"ls"},
			Usage:   "Examine/modify running LiveSessions",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List all LiveSessions",
					Action: func(c *cli.Context) {
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						defer conn.Close()
						client := pb.NewLiveSessionsServiceClient(conn)

						liveSessions, err := client.ListLiveSessions(context.Background(), &empty.Empty{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						lsJSON, _ := json.Marshal(liveSessions)
						fmt.Println(string(lsJSON))

					},
				},
				// TODO (mierdin): Add command to make a session persistent
			},
		},
		{
			Name:    "livelesson",
			Aliases: []string{"ll"},
			Usage:   "Examine/modify running LiveLessons",
			Subcommands: []cli.Command{
				{
					Name:  "kill",
					Usage: "Kill a running livelesson",
					Action: func(c *cli.Context) {
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						llID := pb.LiveLessonId{Id: c.Args().First()}

						result, err := client.KillLiveLesson(context.Background(), &llID)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						if result.Success {
							fmt.Printf("The kill order for livelesson %s was received successfully, and deletion is in progress.\n", llID.Id)
						} else {
							fmt.Println("A problem was encountered processing the livelesson kill order")
						}
						return
					},
				},
				{
					Name:  "list",
					Usage: "List all livelessons",
					Action: func(c *cli.Context) {
						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						liveLessons, err := client.ListLiveLessons(context.Background(), &empty.Empty{})
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						llJSON, _ := json.Marshal(liveLessons)
						fmt.Println(string(llJSON))

					},
				},
				{
					Name:  "create",
					Usage: "Create livelesson(s) from file (TESTING ONLY)",
					Action: func(c *cli.Context) {

						lldef, err := ioutil.ReadFile(c.Args().First())
						if err != nil {
							fmt.Printf("Encountered problem %v\n", err)
							os.Exit(1)
						}

						var lls []pb.LiveLesson

						err = json.Unmarshal([]byte(lldef), &lls)
						if err != nil {
							fmt.Printf("Failed to import %s: %v\n", c.Args().First(), err)
							os.Exit(1)
						}

						conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithInsecure())
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						defer conn.Close()
						client := pb.NewLiveLessonsServiceClient(conn)

						for _, ll := range lls {

							// This command is not meant for production, only testing, and YMMV, but we can at least do a basic
							// sanity check to ensure that the ID field is populated; a sign that the incoming file is at least
							// formatted somewhat correctly
							if ll.ID == "" {
								fmt.Println("Format of incoming file not correct.")
							}

							_, err = client.CreateLiveLesson(context.Background(), &ll)
							if err != nil {
								fmt.Println(err)
								os.Exit(1)
							}
						}

						fmt.Println("OK")
						return
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
