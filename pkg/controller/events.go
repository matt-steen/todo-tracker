package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/rs/zerolog/log"
)

func (c *Controller) handleKeys(evt *tcell.EventKey) *tcell.EventKey {
	key := AsKey(evt)
	if k, ok := c.events[key]; ok {
		c.setErrorText("")

		return k.Action(evt)
	}

	return evt
}

func (c *Controller) handleFormKeys(evt *tcell.EventKey) *tcell.EventKey {
	key := AsKey(evt)
	if k, ok := c.formEvents[key]; ok {
		c.setErrorText("")

		return k.Action(evt)
	}

	return evt
}

func (c *Controller) initEvents() {
	c.events = map[tcell.Key]KeyEvent{}
	c.formEvents = map[tcell.Key]KeyEvent{}

	c.initShowEvents(c.events)
	c.initMoveEvents(c.events)

	c.initFormEvents(c.events)
	c.initLabelEvents(c.events)

	c.initRerankEvents(c.events)
	c.initExitEvent(c.events)

	c.initCancelEvent(c.formEvents)
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

func (c *Controller) getMoveAction(status string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		err := c.db.ChangeStatus(c.ctx, c.selectedTodo, c.selectedStatus, c.db.Statuses[status])
		if err != nil {
			c.setErrorText(err.Error())

			return key
		}

		c.updateTableSelection(status, c.selectedTodo.Rank)

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
		Description: "Move to Closed",
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

func (c *Controller) initFormEvents(events map[tcell.Key]KeyEvent) {
	events[KeyShiftN] = KeyEvent{
		Description: "New Todo",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.titleField.SetText("")
			c.descField.SetText("")

			c.setSelectedTodo(-1, nil)
			c.switchToForm()

			return nil
		},
	}

	events[KeyShiftE] = KeyEvent{
		Description: "Edit Todo",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			if c.selectedTodo == nil {
				log.Debug().Msgf("cannot edit: c.selectedTodo is nil. selectedStatus: %p", c.selectedStatus)

				return key
			}

			c.titleField.SetText(c.selectedTodo.Title)
			c.descField.SetText(c.selectedTodo.Description)

			log.Debug().Msgf("about to edit todo '%s", c.selectedTodo.Title)

			c.switchToForm()

			return nil
		},
	}

	events[KeyShiftU] = KeyEvent{
		Description: "dUplicate Todo",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			if c.selectedTodo == nil {
				log.Debug().Msgf("cannot duplicate: c.selectedTodo is nil. selectedStatus: %p", c.selectedStatus)

				return key
			}

			c.titleField.SetText(c.selectedTodo.Title)
			c.descField.SetText(c.selectedTodo.Description)

			log.Debug().Msgf("about to duplicate todo '%s", c.selectedTodo.Title)

			c.setSelectedTodo(-1, nil)
			c.switchToForm()

			return nil
		},
	}
}

func (c *Controller) initLabelEvents(events map[tcell.Key]KeyEvent) {
	events[KeyShiftL] = KeyEvent{
		Description: "Add Label",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			if c.selectedTodo == nil {
				log.Debug().Msgf("cannot modify labels: c.selectedTodo is nil. selectedStatus: %p", c.selectedStatus)

				return key
			}

			c.addLabel = true
			c.switchToLabelForm()

			return key
		},
	}

	events[KeyShiftR] = KeyEvent{
		Description: "Remove Label",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			if c.selectedTodo == nil {
				log.Debug().Msgf("cannot modify labels: c.selectedTodo is nil. selectedStatus: %p", c.selectedStatus)

				return key
			}

			c.addLabel = false
			c.switchToLabelForm()

			return key
		},
	}
}

func (c *Controller) getRerankAction(direction string) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		var moveFunc func(ctx context.Context, todo *db.Todo) error

		switch direction {
		case "up":
			moveFunc = c.db.MoveUp
		case "down":
			moveFunc = c.db.MoveDown
		case "top":
			moveFunc = c.db.MoveToTop
		case "bottom":
			moveFunc = c.db.MoveToBottom
		}

		err := moveFunc(c.ctx, c.selectedTodo)
		if err != nil {
			c.setErrorText(fmt.Sprintf("error moving %s: %s", direction, err))

			return key
		}

		c.updateTableSelection(c.selectedStatus.Name, c.selectedTodo.Rank)

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

	events[KeyShiftT] = KeyEvent{
		Description: "Shift to Top",
		Action:      c.getRerankAction("top"),
	}

	events[KeyShiftB] = KeyEvent{
		Description: "Shift to Bottom",
		Action:      c.getRerankAction("bottom"),
	}
}

func (c *Controller) initExitEvent(events map[tcell.Key]KeyEvent) {
	events[KeyQ] = KeyEvent{
		Description: "Exit",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.app.Stop()

			log.Info().Msg("exiting application")

			os.Exit(0)

			return key
		},
	}
}

func (c *Controller) initCancelEvent(events map[tcell.Key]KeyEvent) {
	events[tcell.KeyEscape] = KeyEvent{
		Description: "Cancel",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			log.Debug().Msg("cancelling update/creation in progress")

			status := db.StatusClosed
			if c.selectedStatus != nil {
				status = c.selectedStatus.Name
			}

			c.showStatus(status)

			return key
		},
	}
}
