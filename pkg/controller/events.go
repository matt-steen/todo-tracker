package controller

import (
	"github.com/gdamore/tcell/v2"
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

// TODO: should I have constants for the statuses?
func (c *Controller) initShowEvents() {
	c.events[KeyShiftO] = KeyEvent{
		Description: "Show Open",
		Action:      c.getShowAction("open"),
	}

	c.events[KeyShiftC] = KeyEvent{
		Description: "Show Closed",
		Action:      c.getShowAction("closed"),
	}

	c.events[KeyShiftD] = KeyEvent{
		Description: "Show Done",
		Action:      c.getShowAction("done"),
	}

	c.events[KeyShiftH] = KeyEvent{
		Description: "Show On Hold",
		Action:      c.getShowAction("on_hold"),
	}

	c.events[KeyShiftA] = KeyEvent{
		Description: "Show Abandoned",
		Action:      c.getShowAction("abandoned"),
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
		Action:      c.getMoveAction("open"),
	}

	c.events[KeyC] = KeyEvent{
		Description: "Move to Closd",
		Action:      c.getMoveAction("closed"),
	}

	c.events[KeyD] = KeyEvent{
		Description: "Move to Done",
		Action:      c.getMoveAction("done"),
	}

	c.events[KeyH] = KeyEvent{
		Description: "Move to On Hold",
		Action:      c.getMoveAction("on_hold"),
	}

	c.events[KeyA] = KeyEvent{
		Description: "Move to Abandoned",
		Action:      c.getMoveAction("abandoned"),
	}
}
