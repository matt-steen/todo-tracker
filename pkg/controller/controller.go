package controller

import (
	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
)

const (
	descTitleRatio = 2
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

	grid.AddItem(text, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(table, 1, 0, 1, 1, 0, 0, true)

	if err := c.app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}
}

// TODO: is there a way to define the keyboard shortcuts for each status in one place?
// reuse in the actual handling and to display.
func (c *Controller) getHeader(status string) *tview.TextView {
	text := tview.NewTextView()
	text.SetScrollable(false)
	text.SetText("some basic info goes here...")

	return text
}

func (c *Controller) getTable(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false)

	for row, todo := range c.db.Statuses[status].Todos {
		col := 0
		table.SetCell(row, col, tview.NewTableCell(todo.Title).SetExpansion(1).SetReference(todo))
		col++

		table.SetCell(row, col, tview.NewTableCell(todo.Description).SetExpansion(descTitleRatio))
		col++

		labels := ""
		for _, l := range todo.Labels {
			if len(labels) > 0 {
				labels += ", "
			}

			labels += l.Name
		}

		table.SetCell(row, col, tview.NewTableCell(labels).SetTextColor(tcell.ColorGreen).SetExpansion(1))
	}

	table.SetSelectable(true, false)
	table.Select(0, 0)

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			c.app.Stop()
		}
	})

	// progress on adding arbitrary key actions:
	// I'm still not sure how to find the currently selected row in that context...
	//
	// from k9s:
	//
	// func (a *App) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	// 	if k, ok := a.HasAction(ui.AsKey(evt)); ok && !a.Content.IsTopDialog() {
	// 		return k.Action(evt)
	// 	}
	//
	// 	return evt
	// }
	//
	// c.app.SetInputCapture(c.keyboard)

	// TODO: I think this is where I define shortcut key behavior, etc
	/* table.SetSelectedFunc(func(row int, column int) {

	}) */

	return table
}
