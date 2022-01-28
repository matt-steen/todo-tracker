package db_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/matt-steen/todo-tracker/pkg/db"
	"github.com/stretchr/testify/assert"
)

func getDB(assert *assert.Assertions) *db.Database {
	tempFile, err := ioutil.TempFile("/tmp", "test_new_database*")
	assert.Nil(err)

	database, err := db.NewDatabase(context.Background(), tempFile.Name())
	assert.NotNil(database)
	assert.Nil(err)

	return database
}

func addTodo(assert *assert.Assertions, database *db.Database, title, description string) *db.Todo {
	todo, err := database.NewTodo(context.Background(), title, description)
	assert.Nil(err)

	return todo
}

func addDefaultTodo(assert *assert.Assertions, database *db.Database) *db.Todo {
	title := "do some work"
	description := "here are some details of what the work is or where to find out more"

	return addTodo(assert, database, title, description)
}

func TestNewDatabaseBadFile(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database, err := db.NewDatabase(context.Background(), "/alwfkjasfd/asdflkjdsal.sqlite")
	assert.Nil(database)
	assert.NotNil(err)
	assert.Equal("error running base sql: unable to open database file: no such file or directory", err.Error())
}

func TestNewDatabase(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	tempFile, err := ioutil.TempFile("/tmp", "test_new_database")
	assert.Nil(err)

	database, err := db.NewDatabase(context.Background(), tempFile.Name())
	assert.NotNil(database)
	assert.Nil(err)
	database.Close()
}

func TestNewDatabaseIdempotent(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	tempFile, err := ioutil.TempFile("/tmp", "test_new_database*")
	assert.Nil(err)

	database, err := db.NewDatabase(context.Background(), tempFile.Name())
	assert.NotNil(database)
	assert.Nil(err)
	assert.Equal(0, len(database.Todos))
	assert.Equal(5, len(database.Statuses))
	assert.Equal(9, len(database.Labels))

	err = database.Close()
	assert.Nil(err)

	database2, err := db.NewDatabase(context.Background(), tempFile.Name())
	assert.NotNil(database2)
	assert.Nil(err)
	assert.Equal(0, len(database2.Todos))
	assert.Equal(5, len(database2.Statuses))
	assert.Equal(9, len(database2.Labels))

	database2.Close()
}

func TestLoadComplexState(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	ctx := context.Background()

	tempFile, err := ioutil.TempFile("/tmp", "test_new_database*")
	assert.Nil(err)

	database, err := db.NewDatabase(ctx, tempFile.Name())
	assert.Nil(err)

	defer database.Close()

	todo1 := addTodo(assert, database, "todo 1", "")
	todo2 := addTodo(assert, database, "todo 2", "")
	todo3 := addTodo(assert, database, "todo 3", "")

	newLabelName := "busywork"
	label, err := database.NewLabel(ctx, newLabelName)
	assert.Nil(err)

	err = database.AddTodoLabel(ctx, todo1, label)
	assert.Nil(err)

	err = database.AddTodoLabel(ctx, todo1, database.Labels[0])
	assert.Nil(err)

	err = database.ChangeStatus(ctx, todo2, database.Statuses[db.StatusOpen], database.Statuses[db.StatusClosed])
	assert.Nil(err)

	database.Close()

	database2, err := db.NewDatabase(ctx, tempFile.Name())
	assert.Nil(err)

	defer database2.Close()

	assert.Equal(3, len(database2.Todos))
	assert.Equal(2, len(database2.Statuses[db.StatusOpen].Todos))
	assert.Equal(1, len(database2.Statuses[db.StatusClosed].Todos))

	assert.Equal(database2.Statuses[db.StatusOpen].Todos[0].Title, todo1.Title)
	assert.Equal(database2.Statuses[db.StatusOpen].Todos[1].Title, todo3.Title)

	assert.Equal(0, database2.Statuses[db.StatusOpen].Todos[0].Rank)
	assert.Equal(1, database2.Statuses[db.StatusOpen].Todos[1].Rank)

	assert.Equal(database2.Statuses[db.StatusClosed].Todos[0].Title, todo2.Title)
	assert.Equal(0, database2.Statuses[db.StatusClosed].Todos[0].Rank)

	assert.Equal(newLabelName, todo1.Labels[0].Name)
	assert.Equal(database2.Labels[0].Name, todo1.Labels[1].Name)
}

func TestNewLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	name := "busywork"
	label, err := database.NewLabel(context.Background(), name)
	assert.Nil(err)
	assert.Equal(name, label.Name)
}

func TestUpdateLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	ctx := context.Background()

	tempFile, err := ioutil.TempFile("/tmp", "test_new_database*")
	assert.Nil(err)

	database, err := db.NewDatabase(ctx, tempFile.Name())
	assert.Nil(err)

	name := "tag"
	label, err := database.NewLabel(ctx, name)
	assert.Nil(err)
	assert.Equal(name, label.Name)

	name = "heuer"
	err = database.UpdateLabel(ctx, label, name)
	assert.Nil(err)
	assert.Equal(name, label.Name)

	database.Close()

	database2, err := db.NewDatabase(ctx, tempFile.Name())
	assert.Nil(err)

	assert.Equal(name, database2.Labels[len(database2.Labels)-1].Name)
	database2.Close()
}

func TestNewTodo(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	title := "do some work"
	description := "here are some details of what the work is or where to find out more"
	ctx := context.Background()
	todo, err := database.NewTodo(ctx, title, description)
	assert.Nil(err)

	assert.Equal(title, todo.Title)
	assert.Equal(description, todo.Description)

	// confirm that the new todo was added to the end of the list for the open status
	assert.Equal(database.Statuses[db.StatusOpen].Todos[todo.Rank].Title, title)

	todo1, err := database.NewTodo(ctx, "", description)
	assert.Nil(todo1)
	assert.ErrorIs(err, db.ErrEmptyTitle)
}

func TestUpdateTodo(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	ctx := context.Background()

	database := getDB(assert)
	defer database.Close()

	title := "review a proposal"
	description := "it's about something important"
	todo, err := database.NewTodo(ctx, title, description)
	assert.Nil(err)

	assert.Equal(title, todo.Title)
	assert.Equal(description, todo.Description)

	title = "review an important proposal"
	description = "here's a link: https://example.com/something_important"
	err = database.UpdateTodo(ctx, todo, title, description)
	assert.Nil(err)

	assert.Equal(title, todo.Title)
	assert.Equal(description, todo.Description)

	err = database.UpdateTodo(ctx, todo, "", description)
	assert.ErrorIs(err, db.ErrEmptyTitle)
}

func TestAddTodoLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo := addDefaultTodo(assert, database)

	label := database.Labels[0]

	err := database.AddTodoLabel(context.Background(), todo, label)
	assert.Nil(err)

	assert.Equal(label.Name, todo.Labels[0].Name)
}

func TestAddTodoLabelTwice(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo := addDefaultTodo(assert, database)

	label := database.Labels[0]

	err := database.AddTodoLabel(context.Background(), todo, label)
	assert.Nil(err)

	assert.Equal(label.Name, todo.Labels[0].Name)

	err = database.AddTodoLabel(context.Background(), todo, label)
	assert.NotNil(err)
	assert.Equal(
		"error adding label 'task' to todo 'do some work': UNIQUE constraint failed: todo_label.todo_id, todo_label.label_id",
		err.Error(),
	)
}

func TestRemoveTodoLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo := addDefaultTodo(assert, database)

	err := database.AddTodoLabel(context.Background(), todo, database.Labels[0])
	assert.Nil(err)

	err = database.AddTodoLabel(context.Background(), todo, database.Labels[1])
	assert.Nil(err)

	err = database.AddTodoLabel(context.Background(), todo, database.Labels[2])
	assert.Nil(err)

	err = database.RemoveTodoLabel(context.Background(), todo, database.Labels[0])
	assert.Nil(err)

	// confirm preservation of the order of the remaining labels
	assert.Equal(database.Labels[1].Name, todo.Labels[0].Name)
	assert.Equal(database.Labels[2].Name, todo.Labels[1].Name)
}

func TestChangeStatus(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo1 := addTodo(assert, database, "todo 1", "")
	todo2 := addTodo(assert, database, "todo 2", "")
	todo3 := addTodo(assert, database, "todo 3", "")
	todo4 := addTodo(assert, database, "todo 4", "")

	assert.Equal(0, todo1.Rank)
	assert.Equal(1, todo2.Rank)
	assert.Equal(2, todo3.Rank)
	assert.Equal(3, todo4.Rank)

	assert.Equal(todo2.Status, database.Statuses[db.StatusOpen])

	err := database.ChangeStatus(
		context.Background(),
		todo2,
		database.Statuses[db.StatusOpen],
		database.Statuses[db.StatusClosed],
	)
	assert.Nil(err)

	assert.Equal(0, todo2.Rank)
	assert.Equal(1, len(database.Statuses[db.StatusClosed].Todos))
	assert.Equal(database.Statuses[db.StatusClosed], todo2.Status)
	assert.Equal("todo 2", database.Statuses[db.StatusClosed].Todos[0].Title)

	assert.Equal(0, todo1.Rank)
	assert.Equal(1, todo3.Rank)
	assert.Equal(2, todo4.Rank)
	assert.Equal(3, len(database.Statuses[db.StatusOpen].Todos))
}

func initTestChangeStatusErrors(t *testing.T, assert *assert.Assertions) (*db.Database, map[string]*db.Todo) {
	database := getDB(assert)

	todos := map[string]*db.Todo{
		db.StatusOpen:   addTodo(assert, database, "todo open", ""),
		db.StatusClosed: addTodo(assert, database, "todo closed", ""),
		db.StatusOnHold: addTodo(assert, database, "todo on hold", ""),
	}

	ctx := context.Background()

	err := database.ChangeStatus(
		ctx,
		todos[db.StatusClosed],
		database.Statuses[db.StatusOpen],
		database.Statuses[db.StatusClosed],
	)
	assert.Nil(err)

	err = database.ChangeStatus(
		ctx,
		todos[db.StatusOnHold],
		database.Statuses[db.StatusOpen],
		database.Statuses[db.StatusOnHold],
	)
	assert.Nil(err)

	return database, todos
}

func TestChangeStatusErrors(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	cases := []struct {
		name                 string
		oldStatus            string
		newStatus            string
		expectedErrorMessage string
	}{
		{
			name:                 "can't move to yourself",
			oldStatus:            db.StatusOpen,
			newStatus:            db.StatusOpen,
			expectedErrorMessage: db.ErrInvalidTodoMoveNoStatusChange.Error(),
		},
		{
			name:                 "can't move closed to open",
			oldStatus:            db.StatusClosed,
			newStatus:            db.StatusOpen,
			expectedErrorMessage: "cannot move a todo from closed to open",
		},
		{
			name:                 "can't move open to done",
			oldStatus:            db.StatusOpen,
			newStatus:            db.StatusDone,
			expectedErrorMessage: "cannot move a todo from open to done",
		},
		{
			name:                 "can't move on hold to done",
			oldStatus:            db.StatusOnHold,
			newStatus:            db.StatusDone,
			expectedErrorMessage: "cannot move a todo from on_hold to done",
		},
	}

	database, todos := initTestChangeStatusErrors(t, assert)
	defer database.Close()

	for _, testCase := range cases {
		err := database.ChangeStatus(
			context.Background(),
			todos[testCase.oldStatus],
			database.Statuses[testCase.oldStatus],
			database.Statuses[testCase.newStatus],
		)
		assert.NotNil(err)
		assert.Equal(testCase.expectedErrorMessage, err.Error(), testCase.name)
	}
}

func TestChangeStatusValidatesClosedListLimit(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todos := []*db.Todo{}

	for i := 0; i < 6; i++ {
		todo := addTodo(assert, database, fmt.Sprintf("todo %d", i), "")
		todos = append(todos, todo)
	}

	for idx, todo := range todos {
		err := database.ChangeStatus(
			context.Background(),
			todo,
			database.Statuses[db.StatusOpen],
			database.Statuses[db.StatusClosed],
		)

		if idx < 5 {
			assert.Nil(err)
		} else {
			assert.ErrorIs(err, db.ErrMaxClosedTodos)
		}
	}
}

func TestMoveUpTodo(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo1 := addTodo(assert, database, "todo 1", "")
	todo2 := addTodo(assert, database, "todo 2", "")

	assert.Equal(0, todo1.Rank)
	assert.Equal(1, todo2.Rank)

	assert.Equal(todo1.Title, database.Statuses[db.StatusOpen].Todos[0].Title)
	assert.Equal(todo2.Title, database.Statuses[db.StatusOpen].Todos[1].Title)

	err := database.MoveUp(context.Background(), todo1)
	assert.ErrorIs(err, db.ErrCantMoveFirstTodoUp)

	err = database.MoveUp(context.Background(), todo2)
	assert.Nil(err)

	assert.Equal(1, todo1.Rank)
	assert.Equal(0, todo2.Rank)

	assert.Equal(todo2.Title, database.Statuses[db.StatusOpen].Todos[0].Title)
	assert.Equal(todo1.Title, database.Statuses[db.StatusOpen].Todos[1].Title)
}

func TestMoveDownTodo(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	defer database.Close()

	todo1 := addTodo(assert, database, "todo 1", "")
	todo2 := addTodo(assert, database, "todo 2", "")

	assert.Equal(0, todo1.Rank)
	assert.Equal(1, todo2.Rank)

	assert.Equal(todo1.Title, database.Statuses[db.StatusOpen].Todos[0].Title)
	assert.Equal(todo2.Title, database.Statuses[db.StatusOpen].Todos[1].Title)

	err := database.MoveDown(context.Background(), todo2)
	assert.ErrorIs(err, db.ErrCantMoveLastTodoDown)

	err = database.MoveDown(context.Background(), todo1)
	assert.Nil(err)

	assert.Equal(1, todo1.Rank)
	assert.Equal(0, todo2.Rank)

	assert.Equal(todo2.Title, database.Statuses[db.StatusOpen].Todos[0].Title)
	assert.Equal(todo1.Title, database.Statuses[db.StatusOpen].Todos[1].Title)
}
