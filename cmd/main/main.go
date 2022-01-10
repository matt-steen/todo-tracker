package main

import (
	"context"

	"github.com/matt-steen/todo-tracker/pkg/controller"
	"github.com/matt-steen/todo-tracker/pkg/db"
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
	- status (closed, open, on hold, done)
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

	// TODO: handle default and alternate DB locations...
	dbFilename := "/Users/msteen/code/todo-tracker/test.sqlite"

	db, err := db.NewDatabase(ctx, dbFilename)
	if err != nil {
		panic(err)
	}

	controller, err := controller.NewController(db)
	if err != nil {
		panic(err)
	}

	controller.Go()
}
