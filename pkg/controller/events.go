package controller

import (
	"context"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog/log"
)

func (c *Controller) initEvents() {
	c.events = map[tcell.Key]KeyEvent{}
	c.todoEditEvents = map[tcell.Key]KeyEvent{}

	c.initShowEvents(c.events)

	c.initNewEvent(c.events)
	c.initEditEvent(c.events)

	c.initMoveEvents(c.events)

	c.initRerankEvents(c.events)

	c.initExitEvent(c.events)

	c.initCancelEvent(c.todoEditEvents)
}

func (c *Controller) getExitAction() func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		c.app.Stop()

		log.Info().Msg("terminating application")

		os.Exit(0)

		return key
	}
}

func (c *Controller) initExitEvent(events map[tcell.Key]KeyEvent) {
	events[KeyQ] = KeyEvent{
		Description: "Exit",
		Action:      c.getExitAction(),
	}
}

func (c *Controller) getCancelAction() func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		log.Debug().Msg("cancelling update/creation in progress")

		status := db.StatusClosed
		if c.selectedStatus != nil {
			status = c.selectedStatus.Name
		}

		c.showStatus(status)

		return key
	}
}

func (c *Controller) initCancelEvent(events map[tcell.Key]KeyEvent) {
	events[tcell.KeyEscape] = KeyEvent{
		Description: "Cancel",
		Action:      c.getCancelAction(),
	}
}

func (c *Controller) initNewEvent(events map[tcell.Key]KeyEvent) {
	events[KeyN] = KeyEvent{
		Description: "New Todo",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.setSelectedTodo(-1, nil)
			c.switchToForm()

			return key
		},
	}
}

func (c *Controller) getEditAction() func(key *tcell.EventKey) *tcell.EventKey {
	log.Debug().Msgf("in getEditAction. c.selectedTodo: %p", c.selectedTodo)

	return func(key *tcell.EventKey) *tcell.EventKey {
		if c.selectedTodo == nil {
			log.Debug().Msgf("cannot edit: c.selectedTodo is nil. selectedStatus: %p", c.selectedStatus)

			return key
		}

		c.titleField.SetText(c.selectedTodo.Title)
		c.descField.SetText(c.selectedTodo.Description)

		log.Debug().Msgf("about to edit todo '%s", c.selectedTodo.Title)

		c.switchToForm()

		return key
	}
}

func (c *Controller) initEditEvent(events map[tcell.Key]KeyEvent) {
	events[KeyE] = KeyEvent{
		Description: "Edit Todo",
		Action:      c.getEditAction(),
	}
}

func (c *Controller) getShowAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		c.showStatus(status)

		return key
	}
}

func (c *Controller) initShowEvents(events map[tcell.Key]KeyEvent) {
	events[KeyO] = KeyEvent{
		Description: "Show Open",
		Action:      c.getShowAction(db.StatusOpen),
	}

	events[KeyC] = KeyEvent{
		Description: "Show Closed",
		Action:      c.getShowAction(db.StatusClosed),
	}

	events[KeyD] = KeyEvent{
		Description: "Show Done",
		Action:      c.getShowAction(db.StatusDone),
	}

	events[KeyH] = KeyEvent{
		Description: "Show On Hold",
		Action:      c.getShowAction(db.StatusOnHold),
	}

	events[KeyA] = KeyEvent{
		Description: "Show Abandoned",
		Action:      c.getShowAction(db.StatusAbandoned),
	}
}

func (c *Controller) getRerankAction(direction string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		var moveFunc func(ctx context.Context, todo *db.Todo) error
		if direction == "up" {
			moveFunc = c.db.MoveUp
		} else {
			moveFunc = c.db.MoveDown
		}

		err := moveFunc(c.ctx, c.selectedTodo)
		if err != nil {
			log.Error().Err(err).Msgf("error moving %s", direction)

			return key
		}

		// TODO (bug): update the selection if the move was successful

		return key
	}
}

func (c *Controller) initRerankEvents(events map[tcell.Key]KeyEvent) {
	events[KeyShiftK] = KeyEvent{
		Description: "Shift Up",
		Action:      c.getRerankAction("up"),
	}

	events[KeyShiftJ] = KeyEvent{
		Description: "Shift Down",
		Action:      c.getRerankAction("down"),
	}
}

func (c *Controller) getMoveAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		err := c.db.ChangeStatus(c.ctx, c.selectedTodo, c.selectedStatus, c.db.Statuses[status])
		if err != nil {
			// TODO (mvp): how to display the error message to the user here?
			title := "?"
			if c.selectedTodo != nil {
				title = c.selectedTodo.Title
			}

			name := ""
			if c.selectedStatus != nil {
				name = c.selectedStatus.Name
			}

			log.Warn().Err(err).Msgf(
				"error while trying to change status from '%s' to '%s' for todo '%s'.",
				name,
				status,
				title,
			)

			return key
		}

		c.showStatus(status)

		return key
	}
}

func (c *Controller) initMoveEvents(events map[tcell.Key]KeyEvent) {
	events[KeyShiftO] = KeyEvent{
		Description: "Move to Open",
		Action:      c.getMoveAction(db.StatusOpen),
	}

	events[KeyShiftC] = KeyEvent{
		Description: "Move to Closd",
		Action:      c.getMoveAction(db.StatusClosed),
	}

	events[KeyShiftD] = KeyEvent{
		Description: "Move to Done",
		Action:      c.getMoveAction(db.StatusDone),
	}

	events[KeyShiftH] = KeyEvent{
		Description: "Move to On Hold",
		Action:      c.getMoveAction(db.StatusOnHold),
	}

	events[KeyShiftA] = KeyEvent{
		Description: "Move to Abandoned",
		Action:      c.getMoveAction(db.StatusAbandoned),
	}
}
