package db

import (
	"context"
	"database/sql"

	// embed must be imported to allow us to embed base.sql.
	_ "embed"
	"errors"
	"fmt"
	"time"

	// use the sqlite db driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// MaxClosedTodos defines the size of the closed todo list. This is intended to constrict work to items on this list,
// which encourages focus and prioritization.
const MaxClosedTodos = 5

//go:embed base.sql
var baseSQL string

var (
	// ErrMaxClosedTodos is returned from ChangeStatus when attempting to move a todo to the closed list when it is
	// full (i.e., it already has MaxClosedTodos todos).
	ErrMaxClosedTodos = fmt.Errorf(
		"there are already %d closed todos. Complete or abandon something before starting something new",
		MaxClosedTodos,
	)
	// ErrInvalidTodoMoveNoStatusChange is returned from ChangeStatus when the old and new statuses are the same.
	ErrInvalidTodoMoveNoStatusChange = errors.New("cannot move a todo from one status to itself")

	// ErrInvalidTodoMove is returned from ChangeStatus when the old and new statuses are the same.
	ErrInvalidTodoMove = errors.New("cannot move a todo")
	// ErrCantMoveFirstTodoUp is returned from MoveUp when the first todo is moved up.
	ErrCantMoveFirstTodoUp = errors.New("cannot move up the first todo")
	// ErrCantMoveLastTodoDown is returned from MoveDown when the last todo is moved down.
	ErrCantMoveLastTodoDown = errors.New("cannot move down the last todo")
	// ErrNilTodo is returned when a modification is attempted on a nil Todo.
	ErrNilTodo = errors.New("no Todo is currently selected")
	// ErrEmptyTitle is returned when a new or modified todo has no title.
	ErrEmptyTitle = errors.New("Todo title cannot be empty")
)

// Database manages the db connection and the state of the system.
type Database struct {
	conn     *sql.DB
	Statuses map[string]*Status
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
		Statuses: map[string]*Status{},
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

// rollbackOnError attempts to rollback the transaction; if rollback fails, wrap the existing error with information
// on the failed rollback.
func rollbackOnError(tx *sql.Tx, err error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return fmt.Errorf("error rolling back transaction: '%s' after %w", rollbackErr, err)
	}

	return err
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

		d.Statuses[status.Name] = &status
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning statuses: %w", err)
	}

	return nil
}

func (d *Database) loadTodos(ctx context.Context) error {
	log.Debug().Msgf("loading todos from db...")

	todoSQL := `SELECT id, title, description, status_id, rank, created_datetime, updated_datetime
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

		err = rows.Scan(
			&todo.id,
			&todo.Title,
			&todo.Description,
			&statusID,
			&todo.Rank,
			&todo.CreatedDatetime,
			&todo.UpdatedDatetime,
		)
		if err != nil {
			return fmt.Errorf("error scanning todo: %w", err)
		}

		d.Todos = append(d.Todos, &todo)

		for _, status := range d.Statuses {
			if status.id == statusID {
				status.Todos = append(status.Todos, &todo)
				todo.Status = status

				break
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error scanning todos: %w", err)
	}

	for key, status := range d.Statuses {
		for _, todo := range status.Todos {
			log.Debug().Str("status", key).Str("todo", todo.Title).Int("rank", todo.Rank).Msgf("")
		}
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

// NewTodo creates a new Todo with the given title and description; the Todo is added
// at the end of the open list.
func (d *Database) NewTodo(ctx context.Context, title, description string) (*Todo, error) {
	if len(title) == 0 {
		return nil, ErrEmptyTitle
	}

	open := d.Statuses[StatusOpen]

	rank := len(open.Todos)
	now := time.Now()
	todo := &Todo{
		Title:           title,
		Description:     description,
		Labels:          []*Label{},
		Rank:            rank,
		Status:          open,
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

// UpdateTodo updates the Todo with the given title and description.
func (d *Database) UpdateTodo(ctx context.Context, todo *Todo, title, description string) error {
	if todo == nil {
		return ErrNilTodo
	}

	if len(title) == 0 {
		return ErrEmptyTitle
	}

	_, err := d.conn.ExecContext(ctx,
		`UPDATE todo SET title=$1, description=$2 WHERE id=$3`,
		title, description, todo.id,
	)
	if err != nil {
		return fmt.Errorf("error updating todo: %w", err)
	}

	todo.Title = title
	todo.Description = description

	return nil
}

// NewLabel creates a new label with the given name.
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

// UpdateLabel updates the label name.
func (d *Database) UpdateLabel(ctx context.Context, label *Label, name string) error {
	_, err := d.conn.ExecContext(ctx, `UPDATE label SET name=$1 WHERE id=$2`, name, label.id)
	if err != nil {
		return fmt.Errorf("error updating label: %w", err)
	}

	label.Name = name

	return nil
}

func validateStatusChange(todo *Todo, oldStatus, newStatus *Status) error {
	if todo == nil {
		return ErrNilTodo
	}

	if newStatus.Name == StatusClosed && len(newStatus.Todos) >= MaxClosedTodos {
		return ErrMaxClosedTodos
	}

	if newStatus.id == oldStatus.id {
		return ErrInvalidTodoMoveNoStatusChange
	}

	if oldStatus.Name == StatusClosed && newStatus.Name == StatusOpen {
		return fmt.Errorf("%w from %s to %s", ErrInvalidTodoMove, oldStatus.Name, newStatus.Name)
	}

	if (oldStatus.Name == StatusOpen || oldStatus.Name == StatusOnHold) && newStatus.Name == StatusDone {
		return fmt.Errorf("%w from %s to %s", ErrInvalidTodoMove, oldStatus.Name, newStatus.Name)
	}

	return nil
}

func (d *Database) persistStatusChange(ctx context.Context, todo *Todo, oldStatus, newStatus *Status) error {
	txn, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error opening transaction: %w", err)
	}

	_, err = txn.ExecContext(
		ctx,
		`UPDATE todo SET status_id=$1, rank=$2 WHERE id=$3`,
		newStatus.id,
		len(newStatus.Todos),
		todo.id,
	)
	if err != nil {
		return rollbackOnError(txn, fmt.Errorf("error updating todo: %w", err))
	}

	for _, todoToUpdate := range oldStatus.Todos[todo.Rank+1:] {
		log.Debug().Msgf("decrementing rank IN DB for todo %s", todoToUpdate.Title)

		_, err = txn.ExecContext(
			ctx,
			`UPDATE todo SET rank=rank - 1 WHERE id=$1`,
			todoToUpdate.id,
		)
		if err != nil {
			return rollbackOnError(txn, fmt.Errorf("error updating todo rank: %w", err))
		}
	}

	err = txn.Commit()
	if err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}

	return nil
}

func (d *Database) localStatusChange(todo *Todo, oldStatus, newStatus *Status) {
	// don't change objects until after transaction is committed to avoid complexity of reversion if the commit fails
	for _, todoToUpdate := range oldStatus.Todos[todo.Rank+1:] {
		todoToUpdate.Rank--
		log.Debug().Msgf("decremented rank for todo %s with rank %d", todoToUpdate.Title, todoToUpdate.Rank)

		if todoToUpdate.Rank < 0 {
			log.Warn().Msgf("RANK < 0! **************************")
		}
	}

	log.Debug().Int("rank", todo.Rank).Int("len", len(oldStatus.Todos)).Msg("removing todo from oldStatus.Todos")
	oldStatus.Todos = append(oldStatus.Todos[:todo.Rank], oldStatus.Todos[todo.Rank+1:]...)
	log.Debug().Int("rank", todo.Rank).Int("len", len(oldStatus.Todos)).Msg("removed todo from oldStatus.Todos")

	log.Debug().Int("rank", todo.Rank).Int("len", len(newStatus.Todos)).Msg("adding todo to newStatus.Todos")
	newStatus.Todos = append(newStatus.Todos, todo)
	log.Debug().Int("rank", todo.Rank).Int("len", len(newStatus.Todos)).Msg("added todo to newStatus.Todos")

	todo.Status = newStatus
	todo.Rank = len(newStatus.Todos) - 1
	log.Debug().Msgf("setting rank on moved todo to %d", todo.Rank)

	if todo.Rank < 0 {
		log.Warn().Msgf("RANK < 0! **************************")
	}
}

// ChangeStatus moves a Todo from one status to another.
func (d *Database) ChangeStatus(ctx context.Context, todo *Todo, oldStatus, newStatus *Status) error {
	if err := validateStatusChange(todo, oldStatus, newStatus); err != nil {
		return err
	}

	log.Info().Msgf(
		"changing status for todo %s with rank %d in status %s to status %s",
		todo.Title, todo.Rank, oldStatus.Name, newStatus.Name,
	)

	for _, todoToUpdate := range oldStatus.Todos[todo.Rank+1:] {
		log.Debug().Msgf("current rank for todo %s: %d", todoToUpdate.Title, todoToUpdate.Rank)
	}

	if err := d.persistStatusChange(ctx, todo, oldStatus, newStatus); err != nil {
		return err
	}

	d.localStatusChange(todo, oldStatus, newStatus)

	return nil
}

// MoveUp moves a Todo one position up in the list, meaning it reduces the ranking by 1.
// and increases the ranking of the previous Todo.
// If the last Todo is passed, return ErrCantMoveFirstTodoUp.
func (d *Database) MoveUp(ctx context.Context, todo *Todo) error {
	if todo == nil {
		return ErrNilTodo
	}

	if todo.Rank == 0 {
		return ErrCantMoveFirstTodoUp
	}

	todos := todo.Status.Todos

	prevTodo := todos[todo.Rank-1]

	txn, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error opening transaction: %w", err)
	}

	updateRankSQL := `UPDATE todo SET rank=$1 WHERE id=$2`

	_, err = txn.ExecContext(ctx, updateRankSQL, todo.Rank-1, todo.id)
	if err != nil {
		return rollbackOnError(txn, fmt.Errorf("error updating todo: %w", err))
	}

	_, err = txn.ExecContext(ctx, updateRankSQL, prevTodo.Rank+1, prevTodo.id)
	if err != nil {
		return rollbackOnError(txn, fmt.Errorf("error updating todo: %w", err))
	}

	err = txn.Commit()
	if err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}

	todos[todo.Rank-1], todos[todo.Rank] = todos[todo.Rank], todos[todo.Rank-1]

	todo.Rank--
	prevTodo.Rank++

	return nil
}

// MoveDown moves a Todo one position down in the list, meaning it increases the ranking by 1
// and reduces the ranking of the next Todo.
// If the last Todo is passed, return ErrCantMoveLastTodoDown.
func (d *Database) MoveDown(ctx context.Context, todo *Todo) error {
	if todo == nil {
		return ErrNilTodo
	}

	if todo.Rank >= len(todo.Status.Todos)-1 {
		return ErrCantMoveLastTodoDown
	}

	nextTodo := todo.Status.Todos[todo.Rank+1]

	log.Debug().Msgf(
		"calling moveDown for status %s, todo '%s' with rank %d, and nextTodo '%s' with rank %d",
		todo.Status.Name,
		todo.Title,
		todo.Rank,
		nextTodo.Title,
		nextTodo.Rank,
	)

	return d.MoveUp(ctx, nextTodo)
}

// AddTodoLabel adds a Label to a Todo.
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

// RemoveTodoLabel removes a Label from a Todo.
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
