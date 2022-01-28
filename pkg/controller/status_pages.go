package controller

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (c *Controller) getStatusGrid(status string) *tview.Grid {
	header := c.getStatusHeader(status)
	c.statusTables[status] = c.getTable(status)

	grid := tview.NewGrid().SetBorders(true)

	// TODO (low): adjust all headers to take up less space (be consistent!)
	grid.AddItem(header, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(c.statusTables[status], 1, 0, 1, 1, 0, 0, true)

	return grid
}

// getStatusHeader returns the header used for each list of todos.
// it shows the status at the top, followed by 3 columns listing keyboard shortcuts.
// the first column contains misc shortcuts, the second contains "Show <status>" shortcuts,
// and the third contains "Move to <status>" shortcuts. All three columns are sorted alphabetically.
func (c *Controller) getStatusHeader(status string) *tview.Table {
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
