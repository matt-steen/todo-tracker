package controller

import (
	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
)

// StatusContent implements tview.TableContent, which tview.Table uses to update data.
type StatusContent struct {
	tview.TableContentReadOnly
	status *db.Status
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

	if s.status == nil {
		return nil
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
