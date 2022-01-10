package controller

import (
	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
)

// Controller mediates between the model and the view.
type Controller struct {
	db  *db.Database
	app *tview.Application
}

// NewController creates a new Controller to run the app.
func NewController(db *db.Database) (*Controller, error) {
	c := Controller{
		db:  db,
		app: tview.NewApplication(),
	}

	return &c, nil
}

// Go starts the app.
func (c *Controller) Go() {
	grid := tview.NewGrid().SetBorders(true)

	text := c.getHeader("open")
	table := c.getTable("open")

	grid.AddItem(text, 0, 0, 1, 10, 0, 0, false)
	grid.AddItem(table, 1, 0, 1, 10, 0, 0, true)

	if err := c.app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}
}

// TODO: is there a way to define the keyboard shortcuts for each status in one place?
func (c *Controller) getHeader(status string) *tview.TextView {
	text := tview.NewTextView()
	text.SetScrollable(false)
	text.SetText("some basic info goes here...")

	return text
}

func (c *Controller) getTable(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false)

	for row, todo := range c.db.Statuses[status].Todos {
		table.SetCell(row, 0, tview.NewTableCell(todo.Title).SetExpansion(10))
		table.SetCell(row, 1, tview.NewTableCell(todo.Description).SetExpansion(20))
	}

	table.SetSelectable(true, false)
	table.Select(0, 0)

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			c.app.Stop()
		}
	})

	// TODO: I think this is where I define shortcut key behavior, etc
	/* table.SetSelectedFunc(func(row int, column int) {

	}) */

	return table
}
