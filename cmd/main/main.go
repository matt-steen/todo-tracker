package main

import (
	"context"
	"io/fs"
	"os"

	"github.com/matt-steen/todo-tracker/pkg/controller"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

/*

overall design: MVC
model
- sqlite DB:
	- todo
		- title
		- optional description
		- status (FK)
		- created/updated metadata
	- status (closed, open, on hold, done, abandoned)
	- label
		- possible labels (onboarding, high priority/soon, technical, platform, human interaction)
	- todo_labels
		- many:many table
- corresponding lightweight go structs (raw sql)

view
- one tab per status
	- list todos (with labels and descriptions)
	- change order
	- filter by label
	- edit todo
	- change status (make some transitions illegal)
	- open tab only:
		- new todo
	- done tab only:
		- report on "done" items for today (or yesterday)?
	- consistent shortcuts across tabs (only include or list the ones that apply on each tab)

controller
- middle layer to implement changes in model based on input from the view

*/

func main() {
	ctx := context.Background()

	// TODO: handle default and alternate DB locations
	dbFilename := "/Users/msteen/code/todo-tracker/test.sqlite"
	// TODO: handle default and alternate log locations
	logFilename := "/Users/msteen/code/todo-tracker/debug.log"

	// TODO: set default log level to info with the option to set it to debug with a flag?

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
