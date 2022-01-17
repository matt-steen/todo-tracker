package controller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	descTitleRatio = 2
)

// Controller mediates between the model and the view.
type Controller struct {
	ctx            context.Context
	db             *db.Database
	app            *tview.Application
	grid           *tview.Grid
	statusContents map[string]*StatusContent
	selectedTodo   *db.Todo
	selectedStatus *db.Status
	events         map[tcell.Key]KeyEvent
}

// KeyEvent defines an event associated with a keypress.
type KeyEvent struct {
	Description string
	Action      func(*tcell.EventKey) *tcell.EventKey
}

// StatusContent implements tview.TableContent, which tview.Table uses to update data.
type StatusContent struct {
	tview.TableContentReadOnly
	status *db.Status
}

// NewController creates a new Controller to run the app.
func NewController(ctx context.Context, db *db.Database) (*Controller, error) {
	controller := Controller{
		ctx:            ctx,
		db:             db,
		app:            tview.NewApplication(),
		statusContents: map[string]*StatusContent{},
	}

	initKeys()
	controller.initEvents()
	controller.initSignals()

	return &controller, nil
}

func (c *Controller) initSignals() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)

	go func(sig chan os.Signal) {
		<-sig
		os.Exit(0)
	}(sig)
}

// Go starts the app.
func (c *Controller) Go() {
	c.grid = tview.NewGrid().SetBorders(true)

	c.updateStatus("closed")
}

func (c *Controller) updateStatus(status string) {
	header := c.getHeader(status)
	table := c.getTable(status)

	c.grid.Clear()

	c.grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	c.grid.AddItem(table, 1, 0, 1, 1, 0, 0, true)

	c.app.SetInputCapture(c.keyboard)

	// TODO: switch from re-rendering to switching between a limited set of pages
	// the more I switch windows, the longer the stack trace gets when something breaks.
	// that seems like a bad sign...
	//
	// the best I can figure out so far is to use pages and switch between them.
	// then I could also have a page for displaying errors (if I want to go that route)
	//
	// if I use TableContent to give the Table access to the data, then redrawing should happen automatically!
	// create a struct that will hold the list of todos for its status and implement TableContent!
	// use TableContentReadOnly and use the methods we already have to directly manipulate the data structure
	if err := c.app.SetRoot(c.grid, true).SetFocus(c.grid).Run(); err != nil {
		panic(err)
	}
}

func (c *Controller) getHeader(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	row := 0
	table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[yellow]%s", status)))
	row++

	// TODO: control keyboard shortcut ordering in header!
	for key, event := range c.events {
		table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[orange]<%s>[white] %s", tcell.KeyNames[key], event.Description)))
		row++
	}

	return table
}

// when the row selection changes, update the selected Todo.
func (c *Controller) setCurrentRow(row, col int) {
	// adjust for the header row
	if idx := row - 1; idx < len(c.selectedStatus.Todos) && idx >= 0 {
		c.selectedTodo = c.selectedStatus.Todos[idx]
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

	if _, ok := c.statusContents[status]; !ok {
		c.statusContents[status] = &StatusContent{
			status: c.selectedStatus,
		}
	}

	table.SetContent(c.statusContents[status])

	table.SetSelectable(true, false)

	if len(c.selectedStatus.Todos) > 0 {
		table.Select(1, 0).SetFixed(1, 0)

		c.setCurrentRow(1, 0)
	}

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			c.app.Stop()

			log.Info().Msg("terminating application")

			os.Exit(0)
		}
	})

	table.SetSelectionChangedFunc(c.setCurrentRow)

	return table
}

// GetCell returns the cell at the given position or nil if no cell.
func (s *StatusContent) GetCell(row, col int) *tview.TableCell {
	if row == 0 {
		switch col {
		case 0:
			return tview.NewTableCell("title").SetExpansion(1).
				SetTextColor(tcell.ColorYellow).SetSelectable(false)
		case 1:
			return tview.NewTableCell("description").SetExpansion(descTitleRatio).
				SetTextColor(tcell.ColorYellow).SetSelectable(false)
		case 2:
			return tview.NewTableCell("labels").SetExpansion(1).
				SetTextColor(tcell.ColorYellow).SetSelectable(false)
		}
	}

	todo := s.status.Todos[row-1]

	switch col {
	case 0:
		return tview.NewTableCell(todo.Title).SetExpansion(1).SetReference(todo)
	case 1:
		return tview.NewTableCell(todo.Description).SetExpansion(descTitleRatio)
	case 2:
		labels := ""
		for _, l := range todo.Labels {
			if len(labels) > 0 {
				labels += ", "
			}

			labels += l.Name
		}

		return tview.NewTableCell(labels).SetTextColor(tcell.ColorGreen).SetExpansion(1)
	}

	return nil
}

// GetRowCount returns the number of rows in the table.
func (s *StatusContent) GetRowCount() int {
	return len(s.status.Todos) + 1
}

// GetColumnCount returns the number of columns in the table.
func (s *StatusContent) GetColumnCount() int {
	return 3
}
