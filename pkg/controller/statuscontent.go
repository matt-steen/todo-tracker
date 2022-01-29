package controller

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rivo/tview"
)

// labelColors is a list of colors for labels to alternate through so that todos with common labels are easier to spot.
func labelColors() []string {
	return []string{
		"#FF0000",
		"#00FF00",
		"#0000FF",
		"#FFFF00",
		"#FF00FF",
		"#00FFFF",
		"#FFFFFF",
		"#AA0000",
		"#00AA00",
		"#0000AA",
		"#AAAA00",
		"#AA00AA",
		"#00AAAA",
		"#AAAAAA",
	}
}

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
		labels := ""
		for _, l := range todo.Labels {
			if len(labels) > 0 {
				labels += ", "
			}

			colors := labelColors()

			labels += fmt.Sprintf("[%s]%s", colors[l.ID%len(colors)], l.Name)
		}

		return tview.NewTableCell(labels).SetExpansion(1)
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
