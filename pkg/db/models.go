package db

import "time"

type Todo struct {
	id              int
	Title           string
	Description     string
	Labels          []*Label
	Rank            int // starts at 0
	Status          *Status
	CreatedDatetime *time.Time
	UpdatedDatetime *time.Time
}

type Label struct {
	id   int
	Name string
}

type Status struct {
	id    int
	Name  string
	Todos []*Todo
}
