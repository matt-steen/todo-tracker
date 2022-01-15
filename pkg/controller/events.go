package controller

import "github.com/gdamore/tcell/v2"

func (c *Controller) initEvents() {
	c.events = map[tcell.Key]KeyEvent{}

	// TODO: how to surface errors when we actually try to change things?
	// ideally, no logic needs to be duplicated...
	c.events[KeyO] = KeyEvent{
		Description: "Open",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.updateStatus("open")

			return key
		},
	}

	c.events[KeyC] = KeyEvent{
		Description: "Closed",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.updateStatus("closed")

			return key
		},
	}

	c.events[KeyD] = KeyEvent{
		Description: "Done",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.updateStatus("done")

			return key
		},
	}

	c.events[KeyH] = KeyEvent{
		Description: "On Hold",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.updateStatus("on_hold")

			return key
		},
	}

	c.events[KeyA] = KeyEvent{
		Description: "Abandoned",
		Action: func(key *tcell.EventKey) *tcell.EventKey {
			c.updateStatus("abandoned")

			return key
		},
	}
}
