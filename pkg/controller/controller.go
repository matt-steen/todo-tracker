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

// TODO (mvp): create a new todo in the UI
// TODO (mvp): update todo fields
// TODO (mvp): add a label
// TODO (mvp): remove a label
// TODO (mvp): move up/down
// TODO (medium): view recently done tasks (needs more thought)

// Controller mediates between the model and the view.
type Controller struct {
	ctx            context.Context
	db             *db.Database
	app            *tview.Application
	pages          *tview.Pages
	form           *tview.Form
	titleField     *tview.InputField
	descField      *tview.InputField
	tables         map[string]*tview.Table
	statusContents map[string]*StatusContent
	selectedTodo   *db.Todo
	selectedStatus *db.Status
	// events accessible from any status page
	events map[tcell.Key]KeyEvent
	// events accessible only on the todo edit page
	todoEditEvents map[tcell.Key]KeyEvent
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
		tables:         map[string]*tview.Table{},
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
	c.selectedStatus = c.db.Statuses[db.StatusClosed]

	c.pages = c.initPages()

	c.app.SetInputCapture(c.handleKeys)

	if len(c.selectedStatus.Todos) > 0 {
		c.setSelectedTodo(-1, c.selectedStatus.Todos[0])
	}

	if err := c.app.SetRoot(c.pages, true).SetFocus(c.pages).Run(); err != nil {
		panic(err)
	}
}

func pageName(status string) string {
	return fmt.Sprintf("page-%s", status)
}

func (c *Controller) initPages() *tview.Pages {
	pages := tview.NewPages()

	for status := range c.db.Statuses {
		pages.AddPage(pageName(status),
			c.getTableGrid(status),
			true,
			status == db.StatusClosed)
	}

	pages.AddPage(pageName("form"),
		c.getFormGrid(),
		true,
		false)

	return pages
}

func (c *Controller) getTableGrid(status string) *tview.Grid {
	header := c.getHeader(status)
	c.tables[status] = c.getTable(status)

	grid := tview.NewGrid().SetBorders(true)

	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.tables[status], 1, 0, 1, 1, 0, 0, true)

	return grid
}

func (c *Controller) getHeader(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	row := 0
	table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[yellow]%s", status)))
	row++

	// TODO (medium): control keyboard shortcut ordering in header!
	// what order do we want, and how should that information be made available here?
	for key, event := range c.events {
		text := fmt.Sprintf("[orange]<%s>[white] %s", tcell.KeyNames[key], event.Description)
		table.SetCell(row, 0, tview.NewTableCell(text))
		row++
	}

	return table
}

func (c *Controller) getFormGrid() *tview.Grid {
	grid := tview.NewGrid().SetBorders(true)

	// TODO (low): flesh out form header
	header := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	titleMax := 50
	descriptionMax := 500

	c.form = tview.NewForm().
		AddInputField("Title", "", titleMax, nil, nil).
		AddInputField("Description", "", descriptionMax, nil, nil)

	c.titleField, _ = c.form.GetFormItemByLabel("Title").(*tview.InputField)
	c.descField, _ = c.form.GetFormItemByLabel("Description").(*tview.InputField)
	c.form.AddButton("Save", func() {
		var err error
		var todo *db.Todo

		log.Debug().Msgf("saving todo with title '%s'. c.selectedTodo: %p", c.titleField.GetText(), c.selectedTodo)
		if c.selectedTodo == nil {
			todo, err = c.db.NewTodo(c.ctx, c.titleField.GetText(), c.descField.GetText())
		} else {
			err = c.db.UpdateTodo(c.ctx, c.selectedTodo, c.titleField.GetText(), c.descField.GetText())
		}
		if err != nil {
			log.Err(err).Msg("error saving the new todo")

			return
		}

		c.titleField.SetText("")
		c.descField.SetText("")

		// if we don't know where we came from or we created a new todo, then go to open
		if c.selectedStatus == nil || todo != nil {
			c.showStatus(db.StatusOpen)
		} else {
			c.showStatus(c.selectedStatus.Name)
		}

		// TODO (medium): highlight newly added todo after switching

		// TODO (mvp): don't preempt form input for action keys. should I add an explicit cancel button to leave the form?
	})

	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.form, 1, 0, 1, 1, 0, 0, true)

	return grid
}

func (c *Controller) getTodoForRow(row int) *db.Todo {
	// adjust for the header row
	if idx := row - 1; idx < len(c.selectedStatus.Todos) && idx >= 0 {
		return c.selectedStatus.Todos[idx]
	}

	return nil
}

// when the row selection changes, update the selected Todo.
func (c *Controller) setCurrentRow(row, col int) {
	c.setSelectedTodo(row, c.getTodoForRow(row))
}

func (c *Controller) handleKeys(evt *tcell.EventKey) *tcell.EventKey {
	key := AsKey(evt)
	if k, ok := c.events[key]; ok {
		return k.Action(evt)
	}

	return evt
}

func (c *Controller) handleEditKeys(evt *tcell.EventKey) *tcell.EventKey {
	key := AsKey(evt)
	if k, ok := c.todoEditEvents[key]; ok {
		return k.Action(evt)
	}

	return evt
}

func (c *Controller) getTable(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false)

	if _, ok := c.statusContents[status]; !ok {
		c.statusContents[status] = &StatusContent{
			status: c.db.Statuses[status],
		}
	}

	table.SetContent(c.statusContents[status])

	table.SetSelectable(true, false)

	table.SetSelectionChangedFunc(c.setCurrentRow)

	if c.selectedStatus != nil && len(c.selectedStatus.Todos) > 0 {
		table.Select(1, 0).SetFixed(1, 0)
	}

	// TODO (planning): figure out shortcuts and workflow
	// should I use vim-style commands (e.g. :q to quit, :mo to move to open, etc?
	// hit enter to edit or move the todo?
	// something else?
	// there are LOTS of options here...
	/* table.SetSelectedFunc(func (row, col int) {

	})*/

	return table
}

func (c *Controller) setSelectedTodo(row int, todo *db.Todo) {
	c.selectedTodo = todo

	title := "nil"
	if todo != nil {
		title = todo.Title
	}

	name := "nil"
	length := 0

	if c.selectedStatus != nil {
		name = c.selectedStatus.Name
		length = len(c.selectedStatus.Todos)
	}

	log.Debug().
		Str("selectedStatus", name).
		Int("row", row).
		Int("len", length).
		Msgf("setting selectedTodo to '%s'", title)
}

func (c *Controller) showStatus(status string) {
	c.selectedStatus = c.db.Statuses[status]

	c.app.SetInputCapture(c.handleKeys)

	c.pages.SwitchToPage(pageName(status))

	row, _ := c.tables[status].GetSelection()

	if len(c.selectedStatus.Todos) > row-1 && row-1 >= 0 {
		c.setSelectedTodo(row, c.selectedStatus.Todos[row-1])
	} else {
		c.setSelectedTodo(row, nil)
	}
}

// TODO (bug): on edit, e is often appended to focused field

func (c *Controller) switchToForm() {
	c.pages.SwitchToPage(pageName("form"))

	// TODO (bug): when switching to form, focus isn't always on the title
	// restore focus to top row in form
	c.form.SetFocus(0)

	c.app.SetInputCapture(c.handleEditKeys)
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
		// TODO (medium): color-code labels in table (ideally by label id)
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
	if s.status != nil {
		return len(s.status.Todos) + 1
	}

	return 1
}

// GetColumnCount returns the number of columns in the table.
func (s *StatusContent) GetColumnCount() int {
	return 3
}
