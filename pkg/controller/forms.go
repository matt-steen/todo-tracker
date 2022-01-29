package controller

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (c *Controller) switchToForm() {
	title := "New Todo"
	if c.selectedTodo != nil {
		title = "Edit Todo"
	}

	name := "form"

	c.setFormTitle(name, title)

	c.todoForm.SetFocus(0)

	c.pages.SwitchToPage(pageName(name))

	c.app.SetInputCapture(c.handleFormKeys)
}

func (c *Controller) switchToLabelForm() {
	title := "Add Label"
	if !c.addLabel {
		title = "Remove Label"
	}

	name := "labelForm"

	c.setFormTitle(name, title)

	c.updateLabelFormOptions()

	c.labelForm.SetFocus(0)

	c.pages.SwitchToPage(pageName(name))

	c.app.SetInputCapture(c.handleFormKeys)
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
