package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tidwall/buntdb"

	"github.com/urfave/cli/v2"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	app := &cli.App{
		Name:  "roudo",
		Usage: "労働監視くん",
		Commands: []*cli.Command{
			kansiCommand,
			viewCommand,
		},
	}
	return app.Run(os.Args)
}

var kansiCommand = &cli.Command{
	Name:  "kansi",
	Usage: "監視スタート",
	Action: func(c *cli.Context) error {
		db, err := initDB()
		if err != nil {
			panic(err)
		}
		defer db.Close()

		dir, err := getRoudoDir()
		if err != nil {
			return err
		}
		logFile, err := os.OpenFile(filepath.Join(dir, "log.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer logFile.Close()

		logger := slog.New(
			slog.NewJSONHandler(logFile, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}),
		)

		no := &MacNotificator{}

		repo := NewRoudoReportRepository(db)
		reporter := NewRoudoReporter(repo, logger, no)
		mgr := NewRoudoManager(reporter, []IEventWatcher{
			&KeyEventWatcher{},
		}, logger, 1*time.Second)

		return mgr.Kansi()
	},
}

var viewCommand = &cli.Command{
	Name:  "view",
	Usage: "労働時間の一覧を表示",
	Action: func(c *cli.Context) error {
		db, err := initDB()
		if err != nil {
			panic(err)
		}
		defer db.Close()

		viewer := NewViewer(db)
		return viewer.RenderCli(c.Args().First())
	},
}

func initDB() (*buntdb.DB, error) {
	dir, err := getRoudoDir()
	if err != nil {
		return nil, err
	}

	db, err := buntdb.Open(filepath.Join(dir, "roudo.db"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getRoudoDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".roudo")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return "", err
		}
	}
	return dir, nil
}
