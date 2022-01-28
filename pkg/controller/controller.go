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
)

const (
	descTitleRatio = 2
)

// TODO (medium): view recently done tasks (needs more thought)

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

	c.initPages()

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

func (c *Controller) initPages() {
	c.pages = tview.NewPages()

	for status := range c.db.Statuses {
		c.pages.AddPage(pageName(status),
			c.getStatusGrid(status),
			true,
			status == db.StatusClosed)
	}

	c.pages.AddPage(pageName("form"),
		c.getFormGrid(),
		true,
		false)

	c.pages.AddPage(pageName("labelForm"),
		c.getLabelFormGrid(),
		true,
		false)
}
