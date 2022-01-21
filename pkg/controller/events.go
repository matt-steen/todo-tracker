package controller

import (
	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog/log"
)

func (c *Controller) initEvents() {
	c.events = map[tcell.Key]KeyEvent{}

	c.initShowEvents()
	c.initMoveEvents()
}

func (c *Controller) getShowAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		c.showStatus(status)

		return key
	}
}

func (c *Controller) initShowEvents() {
	c.events[KeyShiftO] = KeyEvent{
		Description: "Show Open",
		Action:      c.getShowAction(db.StatusOpen),
	}

	c.events[KeyShiftC] = KeyEvent{
		Description: "Show Closed",
		Action:      c.getShowAction(db.StatusClosed),
	}

	c.events[KeyShiftD] = KeyEvent{
		Description: "Show Done",
		Action:      c.getShowAction(db.StatusDone),
	}

	c.events[KeyShiftH] = KeyEvent{
		Description: "Show On Hold",
		Action:      c.getShowAction(db.StatusOnHold),
	}

	c.events[KeyShiftA] = KeyEvent{
		Description: "Show Abandoned",
		Action:      c.getShowAction(db.StatusAbandoned),
	}
}

func (c *Controller) getMoveAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		err := c.db.ChangeStatus(c.ctx, c.selectedTodo, c.selectedStatus, c.db.Statuses[status])
		if err != nil {
			// TODO: how to display the error message to the user here?
			log.Warn().Err(err).Msgf(
				"error while trying to change status from %s to %s for todo %s.",
				c.selectedStatus.Name,
				status,
				c.selectedTodo.Title,
			)

			return key
		}

		c.showStatus(status)

		return key
	}
}

func (c *Controller) initMoveEvents() {
	c.events[KeyO] = KeyEvent{
		Description: "Move to Open",
		Action:      c.getMoveAction(db.StatusOpen),
	}

	c.events[KeyC] = KeyEvent{
		Description: "Move to Closd",
		Action:      c.getMoveAction(db.StatusClosed),
	}

	c.events[KeyD] = KeyEvent{
		Description: "Move to Done",
		Action:      c.getMoveAction(db.StatusDone),
	}

	c.events[KeyH] = KeyEvent{
		Description: "Move to On Hold",
		Action:      c.getMoveAction(db.StatusOnHold),
	}

	c.events[KeyA] = KeyEvent{
		Description: "Move to Abandoned",
		Action:      c.getMoveAction(db.StatusAbandoned),
	}
}
