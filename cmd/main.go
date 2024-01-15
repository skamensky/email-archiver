package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/skamensky/email-archiver/pkg/client"
	"github.com/skamensky/email-archiver/pkg/database"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/options"
	"github.com/skamensky/email-archiver/pkg/utils"
	"github.com/urfave/cli/v2"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func setup() (models.ClientPool, error) {

	err := godotenv.Load()
	if err != nil {
		return nil, utils.JoinErrors("Error loading .env file", err)
	}

	ops, err := options.New()
	if err != nil {
		return nil, utils.JoinErrors("failed to setup options", err)
	}

	pool := client.NewClientConnPool(ops, nil)

	// make sure we can get a client
	lClient, err := pool.Get()
	if err != nil {
		return nil, utils.JoinErrors("failed to get client", err)
	}
	defer pool.Put(lClient)

	_, err = database.New(ops)
	if err != nil {
		return nil, utils.JoinErrors("failed to setup database", err)
	}

	return pool, nil

}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "list mailboxes",
				Action: func(*cli.Context) error {
					pool, err := setup()
					if err != nil {
						return err
					}
					defer pool.Close()
					mailboxes, err := pool.ListMailboxes()
					if err != nil {
						return err
					}
					for _, mailbox := range mailboxes {
						// TODO: enrich with local state info
						fmt.Println(mailbox.Name())
					}
					return nil
				},
			},
			{
				Name:    "download",
				Aliases: []string{"d"},
				Usage:   "download all mailboxes to a local db",
				Action: func(cCtx *cli.Context) error {
					imapClient, err := setup()
					if err != nil {
						return err
					}
					defer imapClient.Close()
					return imapClient.DownloadAllMailboxes()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
