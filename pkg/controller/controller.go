package controller

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
)

const (
	descTitleRatio = 2
)

// Controller mediates between the model and the view.
type Controller struct {
	db             *db.Database
	app            *tview.Application
	grid           *tview.Grid
	selectedTodo   *db.Todo
	selectedStatus *db.Status
	events         map[tcell.Key]KeyEvent
}

// KeyEvent defines an event associated with a keypress.
type KeyEvent struct {
	Description string
	Action      func(*tcell.EventKey) *tcell.EventKey
}

// NewController creates a new Controller to run the app.
func NewController(db *db.Database) (*Controller, error) {
	c := Controller{
		db:  db,
		app: tview.NewApplication(),
	}

	initKeys()
	c.initEvents()

	return &c, nil
}

// Go starts the app.
func (c *Controller) Go() {
	c.grid = tview.NewGrid().SetBorders(true)

	c.updateStatus("closed")
}

func (c *Controller) updateStatus(status string) {
	text := c.getHeader(status)
	table := c.getTable(status)

	c.grid.Clear()

	c.grid.AddItem(text, 0, 0, 1, 1, 0, 0, false)
	c.grid.AddItem(table, 1, 0, 1, 1, 0, 0, true)

	if err := c.app.SetRoot(c.grid, true).SetFocus(c.grid).Run(); err != nil {
		panic(err)
	}
}

func (c *Controller) getHeader(status string) *tview.TextView {
	text := tview.NewTextView().SetDynamicColors(true)
	text.SetScrollable(false)

	msg := fmt.Sprintf("[yellow]%s\n[white]testing 1 2 3", status)
	text.SetText(msg)

	return text
}

// when the row selection changes, update the selected Todo.
func (c *Controller) setCurrentRow(row, col int) {
	if len(c.selectedStatus.Todos) > row {
		c.selectedTodo = c.selectedStatus.Todos[row]
	}
}

func (c *Controller) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	key := AsKey(evt)
	if k, ok := c.events[key]; ok {
		return k.Action(evt)
	}

	return evt
}

func (c *Controller) getTable(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false)

	c.selectedStatus = c.db.Statuses[status]

	row := 0
	col := 0
	table.SetCell(
		row,
		col,
		tview.NewTableCell("title").SetExpansion(1).SetTextColor(tcell.ColorYellow).SetSelectable(false),
	)
	col++
	table.SetCell(
		row,
		col,
		tview.NewTableCell("description").SetExpansion(1).SetTextColor(tcell.ColorYellow).SetSelectable(false),
	)
	col++
	table.SetCell(
		row,
		col,
		tview.NewTableCell("labels").SetExpansion(1).SetTextColor(tcell.ColorYellow).SetSelectable(false),
	)

	for row, todo := range c.selectedStatus.Todos {
		col := 0
		table.SetCell(row+1, col, tview.NewTableCell(todo.Title).SetExpansion(1).SetReference(todo))
		col++

		table.SetCell(row+1, col, tview.NewTableCell(todo.Description).SetExpansion(descTitleRatio))
		col++

		labels := ""
		for _, l := range todo.Labels {
			if len(labels) > 0 {
				labels += ", "
			}

			labels += l.Name
		}

		table.SetCell(row+1, col, tview.NewTableCell(labels).SetTextColor(tcell.ColorGreen).SetExpansion(1))
	}

	table.SetSelectable(true, false)

	if len(c.selectedStatus.Todos) > 0 {
		table.Select(1, 0).SetFixed(1, 0)
	}

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			c.app.Stop()
		}
	})

	c.app.SetInputCapture(c.keyboard)
	table.SetSelectionChangedFunc(c.setCurrentRow)

	return table
}
