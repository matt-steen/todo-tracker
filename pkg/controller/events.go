package controller

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog/log"
)

func (c *Controller) initEvents() {
	c.events = map[tcell.Key]KeyEvent{}
	c.todoEditEvents = map[tcell.Key]KeyEvent{}

	c.initShowEvents(c.events)
	c.initShowEvents(c.todoEditEvents)

	c.initMoveEvents(c.todoEditEvents)

	c.initExitEvent(c.events)
	c.initExitEvent(c.todoEditEvents)
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

func (c *Controller) getShowAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		c.showStatus(status)

		return key
	}
}

func (c *Controller) initShowEvents(events map[tcell.Key]KeyEvent) {
	events[KeyShiftO] = KeyEvent{
		Description: "Show Open",
		Action:      c.getShowAction(db.StatusOpen),
	}

	events[KeyShiftC] = KeyEvent{
		Description: "Show Closed",
		Action:      c.getShowAction(db.StatusClosed),
	}

	events[KeyShiftD] = KeyEvent{
		Description: "Show Done",
		Action:      c.getShowAction(db.StatusDone),
	}

	events[KeyShiftH] = KeyEvent{
		Description: "Show On Hold",
		Action:      c.getShowAction(db.StatusOnHold),
	}

	events[KeyShiftA] = KeyEvent{
		Description: "Show Abandoned",
		Action:      c.getShowAction(db.StatusAbandoned),
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

			log.Warn().Err(err).Msgf(
				"error while trying to change status from %s to %s for todo %s.",
				c.selectedStatus.Name,
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
	events[KeyO] = KeyEvent{
		Description: "Move to Open",
		Action:      c.getMoveAction(db.StatusOpen),
	}

	events[KeyC] = KeyEvent{
		Description: "Move to Closd",
		Action:      c.getMoveAction(db.StatusClosed),
	}

	events[KeyD] = KeyEvent{
		Description: "Move to Done",
		Action:      c.getMoveAction(db.StatusDone),
	}

	events[KeyH] = KeyEvent{
		Description: "Move to On Hold",
		Action:      c.getMoveAction(db.StatusOnHold),
	}

	events[KeyA] = KeyEvent{
		Description: "Move to Abandoned",
		Action:      c.getMoveAction(db.StatusAbandoned),
	}
}
