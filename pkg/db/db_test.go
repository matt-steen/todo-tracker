package db_test

import (
	"context"
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

func addTodo(assert *assert.Assertions, database *db.Database) *db.Todo {
	title := "do some work"
	description := "here are some details of what the work is or where to find out more"
	todo, err := database.NewTodo(context.Background(), title, description)
	assert.Nil(err)

	return todo
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
}

func TestNewLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)

	name := "busywork"
	label, err := database.NewLabel(context.Background(), name)
	assert.Nil(err)
	assert.Equal(name, label.Name)
}

func TestNewTodo(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)

	title := "do some work"
	description := "here are some details of what the work is or where to find out more"
	todo, err := database.NewTodo(context.Background(), title, description)
	assert.Nil(err)

	assert.Equal(title, todo.Title)
	assert.Equal(description, todo.Description)

	// confirm that the new todo was added to the end of the list for the open status
	for _, s := range database.Statuses {
		if s.Name == "open" {
			assert.Equal(s.Todos[todo.Rank].Title, title)

			break
		}
	}
}

func TestAddTodoLabel(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	todo := addTodo(assert, database)

	label := database.Labels[0]

	err := database.AddTodoLabel(context.Background(), todo, label)
	assert.Nil(err)

	assert.Equal(label.Name, todo.Labels[0].Name)
}

func TestAddTodoLabelTwice(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	database := getDB(assert)
	todo := addTodo(assert, database)

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
	todo := addTodo(assert, database)

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

// TODO: setup something moderately complicated and then reload it to fully verify db init
