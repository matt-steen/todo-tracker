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

// TODO (medium): view recently done tasks (needs more thought)

// TODO (mvp): organize functions in controller.go

// TODO (mvp): how to display error messages?

// Controller mediates between the model and the view.
type Controller struct {
	ctx context.Context
	db  *db.Database
	app *tview.Application

	// selectedStatus contains the most recently selected Status, which is needed to return when escaping from a form,
	// among other cases.
	selectedStatus *db.Status
	// selectedTodo contains the currently selected Todo object that will be acted upon by shortcut keys. It may be nil!
	selectedTodo *db.Todo

	// Controller maintains programatically named pages that the user can switch between.
	// Importantly, the contents of each page exist even when not visible.
	// There's one page for each status, where we display the Todos with that status,
	// one page with a basic form to add or edit Todos, and one page with a form to add or remove Labels from a Todo.
	pages *tview.Pages

	// statusTables stores one table per status; these are the visible table objects that contain the Todos and a
	// header row.
	statusTables map[string]*tview.Table

	formHeaderTables map[string]*tview.Table

	// The todoForm contains fields for the title and description and a save button.
	todoForm   *tview.Form
	titleField *tview.InputField
	descField  *tview.InputField

	// The labelForm contains a dropdown that lists either Labels that do or do not currently apply to the selectedTodo
	// depending on whether we are adding or removing Labels. It also contains a save button.
	labelForm     *tview.Form
	labelDropDown *tview.DropDown
	// addLabel indicates whether we are currently adding or removing a label
	addLabel bool

	// events contains a map of keyboard actions accessible from status pages
	events map[tcell.Key]KeyEvent
	// formEvents contains a map of keyboard actions accessible from form pages
	formEvents map[tcell.Key]KeyEvent
}

// KeyEvent defines an event associated with a keypress.
type KeyEvent struct {
	Description string
	Action      func(*tcell.EventKey) *tcell.EventKey
}

// NewController creates a new Controller to run the app.
func NewController(ctx context.Context, db *db.Database) (*Controller, error) {
	controller := Controller{
		ctx:              ctx,
		db:               db,
		app:              tview.NewApplication(),
		statusTables:     map[string]*tview.Table{},
		formHeaderTables: map[string]*tview.Table{},
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

	pages.AddPage(pageName("labelForm"),
		c.getLabelFormGrid(),
		true,
		false)

	return pages
}

func (c *Controller) getTableGrid(status string) *tview.Grid {
	header := c.getHeader(status)
	c.statusTables[status] = c.getTable(status)

	grid := tview.NewGrid().SetBorders(true)

	// TODO (low): adjust all headers to take up less space (be consistent!)
	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.statusTables[status], 1, 0, 1, 1, 0, 0, true)

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

	name := "form"

	c.initFormHeader(name)
	c.initForm()

	grid.AddItem(c.formHeaderTables[name], 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.todoForm, 1, 0, 1, 1, 0, 0, true)

	return grid
}

func (c *Controller) getLabelFormGrid() *tview.Grid {
	grid := tview.NewGrid().SetBorders(true)

	name := "labelForm"

	c.initFormHeader(name)
	c.initLabelForm()

	grid.AddItem(c.formHeaderTables[name], 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.labelForm, 1, 0, 1, 1, 0, 0, true)

	return grid
}

func (c *Controller) setFormTitle(tableName, title string) {
	c.formHeaderTables[tableName].SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("[yellow]%s", title)))
}

func (c *Controller) initFormHeader(name string) {
	c.formHeaderTables[name] = tview.NewTable().SetBorders(false).SetSelectable(false, false)
	row := 1

	for key, event := range c.formEvents {
		text := fmt.Sprintf("[orange]<%s>[white] %s", tcell.KeyNames[key], event.Description)
		c.formHeaderTables[name].SetCell(row, 0, tview.NewTableCell(text))
		row++
	}
}

func (c *Controller) initForm() {
	titleMax := 50
	descriptionMax := 500

	c.todoForm = tview.NewForm().
		AddInputField("Title", "", titleMax, nil, nil).
		AddInputField("Description", "", descriptionMax, nil, nil)

	c.titleField, _ = c.todoForm.GetFormItemByLabel("Title").(*tview.InputField)
	c.descField, _ = c.todoForm.GetFormItemByLabel("Description").(*tview.InputField)
	c.todoForm.AddButton("Save", func() {
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

		var rank int
		// if we don't know where we came from or we created a new todo, then go to open
		status := db.StatusOpen
		if c.selectedStatus != nil && todo == nil {
			status = c.selectedStatus.Name
			rank = c.selectedTodo.Rank
		} else {
			rank = todo.Rank
		}

		// select the new/edited todo and return to the todo list for its status
		c.updateTableSelection(status, rank)
		c.showStatus(status)
	})
}

func (c *Controller) updateLabelFormOptions() {
	options := []string{}

	for _, label := range c.db.Labels {
		found := false

		for _, todoLabel := range c.selectedTodo.Labels {
			if todoLabel.Name == label.Name {
				found = true

				break
			}
		}

		if (found && !c.addLabel) || (!found && c.addLabel) {
			options = append(options, label.Name)
		}
	}

	c.labelDropDown.SetOptions(options, nil)
	c.labelDropDown.SetCurrentOption(-1)
}

func (c *Controller) getSelectedLabel() *db.Label {
	_, name := c.labelDropDown.GetCurrentOption()

	for _, label := range c.db.Labels {
		if label.Name == name {
			return label
		}
	}

	log.Error().Msgf("no label found with name '%s'", name)

	return nil
}

func (c *Controller) initLabelForm() {
	c.labelForm = tview.NewForm().
		AddDropDown("Label", []string{}, -1, nil)

	c.labelDropDown, _ = c.labelForm.GetFormItemByLabel("Label").(*tview.DropDown)

	c.labelForm.AddButton("Save", func() {
		label := c.getSelectedLabel()

		if c.addLabel {
			log.Debug().Msgf("adding label '%s' to todo '%s'", label.Name, c.selectedTodo.Title)
			if err := c.db.AddTodoLabel(c.ctx, c.selectedTodo, label); err != nil {
				log.Error().Msgf("error adding label: %s", err)
			}
		} else {
			log.Debug().Msgf("removing label '%s' to todo '%s'", label.Name, c.selectedTodo.Title)
			if err := c.db.RemoveTodoLabel(c.ctx, c.selectedTodo, label); err != nil {
				log.Error().Msgf("error removing label: %s", err)
			}
		}

		c.showStatus(c.selectedStatus.Name)
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
	if k, ok := c.formEvents[key]; ok {
		return k.Action(evt)
	}

	return evt
}

func (c *Controller) getTable(status string) *tview.Table {
	table := tview.NewTable().SetBorders(false)

	statusContent := &StatusContent{
		status: c.db.Statuses[status],
	}

	table.SetContent(statusContent)

	table.SetSelectable(true, false)

	table.SetSelectionChangedFunc(c.setCurrentRow)

	if c.selectedStatus != nil && len(c.selectedStatus.Todos) > 0 {
		table.Select(1, 0).SetFixed(1, 0)
	}

	return table
}

// updateTableSelection updates the selection for the table matching the given status to keep it
// in sync with recently taken actions, e.g. when moving a Todo up or down.
func (c *Controller) updateTableSelection(status string, rank int) {
	if c.statusTables[status].GetRowCount() > rank {
		c.statusTables[status].Select(rank+1, 0)
	} else {
		log.Warn().Msgf("couldn't select; rank was too high: %d (row count: %d)", rank, c.statusTables[status].GetRowCount())
	}
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

	row, _ := c.statusTables[status].GetSelection()

	length := len(c.selectedStatus.Todos)

	if length > row-1 && row-1 >= 0 {
		c.setSelectedTodo(row, c.selectedStatus.Todos[row-1])
	} else if length > 0 {
		c.setSelectedTodo(length, c.selectedStatus.Todos[length-1])
	} else {
		c.setSelectedTodo(-1, nil)
	}

	if c.selectedStatus != nil && c.selectedTodo != nil {
		c.updateTableSelection(c.selectedStatus.Name, c.selectedTodo.Rank)
	}

	c.pages.SwitchToPage(pageName(status))
}

func (c *Controller) switchToForm() {
	title := "New Todo"
	if c.selectedTodo != nil {
		title = "Edit Todo"
	}

	c.setFormTitle("form", title)

	c.todoForm.SetFocus(0)

	c.pages.SwitchToPage(pageName("form"))

	c.app.SetInputCapture(c.handleEditKeys)
}

func (c *Controller) switchToLabelForm() {
	title := "Add Label"
	if !c.addLabel {
		title = "Remove Label"
	}

	c.setFormTitle("labelForm", title)

	c.updateLabelFormOptions()

	c.labelForm.SetFocus(0)

	c.pages.SwitchToPage(pageName("labelForm"))

	c.app.SetInputCapture(c.handleEditKeys)
}
