package main

import (
	"context"
	"io/fs"
	"os"
	"os/user"
	"path"

	"github.com/matt-steen/todo-tracker/pkg/controller"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	user, _ := user.Current()

	dbFilename, ok := os.LookupEnv("TT_DB_FILENAME")
	if !ok {
		dbFilename = path.Join(user.HomeDir, ".todo_tracker.sqlite")
	}

	logFilename, ok := os.LookupEnv("TT_LOG_FILENAME")
	if !ok {
		logFilename = path.Join(user.HomeDir, ".todo_tracker.log")
	}

	// TODO (low): set default log level to info with the option to set it to debug with a flag?

	filePerms := 0o666

	logFile, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, fs.FileMode(filePerms))
	if err != nil {
		panic(err)
	}

	defer logFile.Close()

	log.Logger = log.With().Caller().Logger().Output(zerolog.ConsoleWriter{
		Out: logFile, TimeFormat: "2006-01-02_15:04:05",
	})

	log.Info().Msg("starting application...")

	db, err := db.NewDatabase(ctx, dbFilename)
	if err != nil {
		panic(err)
	}

	controller, err := controller.NewController(ctx, db)
	if err != nil {
		panic(err)
	}

	controller.Go()
}
