package main

import (
	"github.com/rivo/tview"
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
- how to conceptualize this?

*/

func main() {
	// TODO: how to have shortcuts that act on the current item rather than switching?
	// TODO: organization: use multiple tabs
	app := tview.NewApplication()
	list := tview.NewList().
		AddItem("List item 1", "Some explanatory text", 'a', nil).
		AddItem("List item 2", "Some explanatory text", 'b', nil).
		AddItem("List item 3", "Some explanatory text", 'c', nil).
		AddItem("List item 4", "Some explanatory text", 'd', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})

	if err := app.SetRoot(list, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}
}
