package controller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	descTitleRatio = 2
)

// TODO (mvp): add a label
// TODO (mvp): remove a label
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

// getHeader returns the header used for each list of todos.
// it shows the status at the top, followed by 3 columns listing keyboard shortcuts.
// the first column contains misc shortcuts, the second contains "Show <status>" shortcuts,
// and the third contains "Move to <status>" shortcuts. All three columns are sorted alphabetically.
func (c *Controller) getHeader(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	row := 0
	table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[yellow]%s", status)))
	row++

	shortcuts := map[int][]string{
		0: {},
		1: {},
		2: {},
	}

	for key, event := range c.events {
		text := fmt.Sprintf("[orange]<%s>[white] %s", tcell.KeyNames[key], event.Description)

		switch event.Description[:4] {
		case "Show":
			shortcuts[1] = append(shortcuts[1], text)
		case "Move":
			shortcuts[2] = append(shortcuts[2], text)
		default:
			shortcuts[0] = append(shortcuts[0], text)
		}
	}

	for col := 0; col < 3; col++ {
		sort.Strings(shortcuts[col])
	}

	for row-1 < len(shortcuts[0]) || row-1 < len(shortcuts[1]) {
		for col := 0; col < 3; col++ {
			if row-1 < len(shortcuts[col]) {
				table.SetCell(row, col, tview.NewTableCell(shortcuts[col][row-1]).SetExpansion(1))
			}
		}

		row++
	}

	return table
}

func (c *Controller) getFormGrid() *tview.Grid {
	grid := tview.NewGrid().SetBorders(true)

	c.initFormHeader()
	c.initForm()

	grid.AddItem(c.tables["form"], 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.form, 1, 0, 1, 1, 0, 0, true)

	return grid
}

func (c *Controller) setFormTitle() {
	action := "New Todo"
	if c.selectedTodo != nil {
		action = "Edit Todo"
	}

	c.tables["form"].SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("[yellow]%s", action)))
}

func (c *Controller) initFormHeader() {
	c.tables["form"] = tview.NewTable().SetBorders(false).SetSelectable(false, false)
	row := 1

	for key, event := range c.todoEditEvents {
		text := fmt.Sprintf("[orange]<%s>[white] %s", tcell.KeyNames[key], event.Description)
		c.tables["form"].SetCell(row, 0, tview.NewTableCell(text))
		row++
	}
}

func (c *Controller) initForm() {
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

		// TODO (mvp): highlight newly added todo after switching
	})
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

// updateTableSelection updates the selection for the table matching the given status to keep it
// in sync with recently taken actions, e.g. when moving a Todo up or down.
func (c *Controller) updateTableSelection(status string, rank int) {
	c.tables[status].Select(rank+1, 0)
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

func (c *Controller) switchToForm() {
	c.setFormTitle()

	c.form.SetFocus(0)

	c.pages.SwitchToPage(pageName("form"))

	c.app.SetInputCapture(c.handleEditKeys)
}
