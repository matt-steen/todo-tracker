package db

import "time"

// These constants refer to the statuses supported by the app.
const (
	StatusClosed    = "closed"
	StatusOpen      = "open"
	StatusDone      = "done"
	StatusOnHold    = "on_hold"
	StatusAbandoned = "abandoned"
)

// Todo contains individual todo entries and associated labels from the todo_labels table.
type Todo struct {
	id          int
	Title       string
	Description string
	Labels      []*Label
	// Rank is maintained within each status. It starts at 0 and increments by 1.
	// When a Todo is moved to a different status, it is appended to the list, so it has the
	// highest rank in that list.
	Rank            int
	Status          *Status
	CreatedDatetime *time.Time
	UpdatedDatetime *time.Time
}

// Label contains labels that can be applied to todos.
type Label struct {
	id   int
	Name string
}

// Status represents a status entry and contains pointers to associated Todos.
type Status struct {
	id    int
	Name  string
	Todos []*Todo
}
