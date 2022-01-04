package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	// use the sqlite db driver.
	_ "github.com/mattn/go-sqlite3"
)

//go:embed base.sql
var baseSQL string

// Database manages the db connection and the state of the system.
type Database struct {
	conn     *sql.DB
	Statuses []*Status
	Labels   []*Label
	Todos    []*Todo
}

// NewDatabase connects to the sqlite database at the given filename, initializes the structure
// if not present, and loads existing data into memory.
func NewDatabase(ctx context.Context, filename string) (*Database, error) {
	conn, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, fmt.Errorf("error connecting to sqlite db at %s: %w", filename, err)
	}

	database := Database{
		conn:     conn,
		Statuses: []*Status{},
		Labels:   []*Label{},
		Todos:    []*Todo{},
	}

	err = database.initialize(ctx)
	if err != nil {
		return nil, err
	}

	err = database.loadData(ctx)
	if err != nil {
		return nil, err
	}

	return &database, nil
}

func (d *Database) initialize(ctx context.Context) error {
	// run idempotent setup sql to create empty tables if they don't exist
	if _, err := d.conn.ExecContext(ctx, baseSQL); err != nil {
		return fmt.Errorf("error running base sql: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	if err := d.conn.Close(); err != nil {
		return fmt.Errorf("error closing db: %w", err)
	}

	return nil
}

func (d *Database) loadData(ctx context.Context) error {
	var err error

	err = d.loadLabels(ctx)
	if err != nil {
		return err
	}

	err = d.loadStatuses(ctx)
	if err != nil {
		return err
	}

	err = d.loadTodos(ctx)
	if err != nil {
		return err
	}

	err = d.loadTodoLabels(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) loadLabels(ctx context.Context) error {
	labelSQL := `SELECT id, name FROM label`

	rows, err := d.conn.QueryContext(ctx, labelSQL)
	if err != nil {
		return fmt.Errorf("error loading labels: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var label Label

		err = rows.Scan(&label.id, &label.Name)
		if err != nil {
			return fmt.Errorf("error scanning label: %w", err)
		}

		d.Labels = append(d.Labels, &label)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning labels: %w", err)
	}

	return nil
}

func (d *Database) loadStatuses(ctx context.Context) error {
	statusSQL := `SELECT id, name FROM status`

	rows, err := d.conn.QueryContext(ctx, statusSQL)
	if err != nil {
		return fmt.Errorf("error loading statuses: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var status Status

		err = rows.Scan(&status.id, &status.Name)
		if err != nil {
			return fmt.Errorf("error scanning status: %w", err)
		}

		d.Statuses = append(d.Statuses, &status)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning statuses: %w", err)
	}

	return nil
}

func (d *Database) loadTodos(ctx context.Context) error {
	todoSQL := `SELECT id, title, description, status_id, created_datetime, updated_datetime
				FROM todo
				ORDER BY status_id, rank`

	rows, err := d.conn.QueryContext(ctx, todoSQL)
	if err != nil {
		return fmt.Errorf("error loading todos: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var todo Todo

		var statusID int

		err = rows.Scan(&todo.id, &todo.Title, &todo.Description, &statusID, &todo.CreatedDatetime, &todo.UpdatedDatetime)
		if err != nil {
			return fmt.Errorf("error scanning todo: %w", err)
		}

		d.Todos = append(d.Todos, &todo)

		for _, status := range d.Statuses {
			if status.id == statusID {
				status.Todos = append(status.Todos, &todo)

				break
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning todos: %w", err)
	}

	return nil
}

func (d *Database) loadTodoLabels(ctx context.Context) error {
	todoSQL := `SELECT todo_id, label_id
				FROM todo_label
				ORDER BY todo_id, label_id`

	rows, err := d.conn.QueryContext(ctx, todoSQL)
	if err != nil {
		return fmt.Errorf("error loading todos: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var todoID int

		var labelID int

		err = rows.Scan(&todoID, &labelID)
		if err != nil {
			return fmt.Errorf("error scanning todo-label: %w", err)
		}

		var label *Label

		for _, l := range d.Labels {
			if l.id == labelID {
				label = l

				break
			}
		}

		for _, todo := range d.Todos {
			if todo.id == todoID {
				todo.Labels = append(todo.Labels, label)

				break
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning todo-labels: %w", err)
	}

	return nil
}

// NewTodo creates a new todo with the given title and description; the todo is added
// at the end of the open list.
func (d *Database) NewTodo(ctx context.Context, title, description string) (*Todo, error) {
	var open *Status

	for _, status := range d.Statuses {
		if status.Name == "open" {
			open = status

			break
		}
	}

	rank := len(open.Todos)
	now := time.Now()
	todo := &Todo{
		Title:           title,
		Description:     description,
		Labels:          []*Label{},
		Rank:            rank,
		CreatedDatetime: &now,
		UpdatedDatetime: &now,
	}

	result, err := d.conn.ExecContext(ctx,
		`INSERT INTO todo (title, description, status_id, rank, created_datetime, updated_datetime) 
		     VALUES ($1, $2, $3, $4, $5, $6)`,
		todo.Title, todo.Description, open.id, todo.Rank, todo.CreatedDatetime, todo.UpdatedDatetime,
	)
	if err != nil {
		return nil, fmt.Errorf("error adding todo: %w", err)
	}

	todoID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error getting id of new todo %s: %w", title, err)
	}

	open.Todos = append(open.Todos, todo)
	todo.id = int(todoID)

	return todo, nil
}

func (d *Database) NewLabel(ctx context.Context, name string) (*Label, error) {
	result, err := d.conn.ExecContext(ctx, `INSERT INTO label (name) VALUES ($1)`, name)
	if err != nil {
		return nil, fmt.Errorf("error adding label %s: %w", name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error getting id of new label %s: %w", name, err)
	}

	label := &Label{id: int(id), Name: name}
	d.Labels = append(d.Labels, label)

	return label, nil
}

func (d *Database) ChangeStatus(ctx context.Context, todo *Todo, status *Status) error { // TODO
	// tx, err := d.conn.BeginTx(ctx, nil)
	// DB:
	// update todo status_id and rank
	// update rank of anything behind this one in the list -> need to operate in a transaction!
	//
	// Go objects:
	// move todo from current status to new status (bottom of the list)
	//     error if closed is already full! should that also be an error on db load?
	// update todo status_id and rank
	// update ranks of anything behind this one in the old list
	return nil
}

func (d *Database) MoveUp(ctx context.Context) error { // TODO
	// DB:
	// update rank for this todo and whatever is above it (in a transaction)
	//
	// Go objects:
	// update rank for this todo and whatever is above it
	return nil
}

func (d *Database) MoveDown(ctx context.Context) error { // TODO
	// model on MoveUp...
	return nil
}

func (d *Database) AddTodoLabel(ctx context.Context, todo *Todo, label *Label) error {
	_, err := d.conn.ExecContext(ctx,
		`INSERT INTO todo_label (todo_id, label_id) VALUES ($1, $2)`,
		todo.id, label.id,
	)
	if err != nil {
		return fmt.Errorf("error adding label '%s' to todo '%s': %w", label.Name, todo.Title, err)
	}

	todo.Labels = append(todo.Labels, label)

	return nil
}

func (d *Database) RemoveTodoLabel(ctx context.Context, todo *Todo, label *Label) error {
	_, err := d.conn.ExecContext(ctx,
		`DELETE FROM todo_label WHERE todo_id = $1 AND label_id = $2`,
		todo.id, label.id,
	)
	if err != nil {
		return fmt.Errorf("error removing label '%s' from todo '%s': %w", label.Name, todo.Title, err)
	}

	// remove the label from the list
	for i, l := range todo.Labels {
		if l.id == label.id {
			todo.Labels = append(todo.Labels[:i], todo.Labels[i+1:]...)

			break
		}
	}

	return nil
}
