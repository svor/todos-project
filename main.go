package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type Todo struct {
	ID        int    `json:"_id,omitempty" bson:"_id,omitempty"`
	Completed bool   `json:"completed" bson:"completed"`
	Body      string `json:"body" bson:"body"`
}

func main() {
	fmt.Println("Hello, World!")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	// POSTGRES_URI := os.Getenv("POSTGRES_URI")
	POSTGRES_URI := "postgresql://user:password@db:5432/todos"

	connection, err := pgx.Connect(context.Background(), POSTGRES_URI)
	if err != nil {
		log.Fatal("Error connecting to database", err)
	}
	defer connection.Close(context.Background())

	// Create the table if it doesn't exist
	err = createTableIfNotExists(connection)
	if err != nil {
		log.Fatal("Error creating table", err)
	}

	err = connection.Ping(context.Background())
	if err != nil {
		log.Fatal("Error pinging database", err)

	}

	fmt.Println("Connected to database")

	app := fiber.New()

	app.Get("/api/todos", func(c *fiber.Ctx) error {
		return getTodos(c, connection)
	})
	app.Post("/api/todos", func(c *fiber.Ctx) error {
		return createTodo(c, connection)
	})
	app.Patch("/api/todos/:id", func(c *fiber.Ctx) error {
		return updateTodo(c, connection)
	})
	app.Delete("/api/todos/:id", func(c *fiber.Ctx) error {
		return deleteTodo(c, connection)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

// Create the todos table if it doesn't exist
func createTableIfNotExists(connection *pgx.Conn) error {
	// SQL query to create the todos table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS todos (
		id SERIAL PRIMARY KEY,
		completed BOOLEAN NOT NULL,
		body TEXT NOT NULL
	);
	`

	// Execute the query to create the table
	_, err := connection.Exec(context.Background(), createTableSQL)
	if err != nil {
		return fmt.Errorf("could not create todos table: %w", err)
	}

	return nil
}

func getTodos(c *fiber.Ctx, connection *pgx.Conn) error {
	var todos []Todo

	rows, err := connection.Query(context.Background(), "SELECT id, completed, body FROM todos")

	if err != nil {
		log.Fatal("Error querying database", err)
	}
	defer rows.Close()

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Completed, &todo.Body)
		if err != nil {
			log.Fatal("Error scanning row", err)
		}
		todos = append(todos, todo)
	}

	if rows.Err() != nil {
		log.Fatal("Error iterating rows", err)
	}

	return c.JSON(todos)
}

func createTodo(c *fiber.Ctx, connection *pgx.Conn) error {
	todo := new(Todo)

	err := c.BodyParser(todo)
	if err != nil {
		return err
	}

	if todo.Body == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Body is required",
		})
	}

	var insertedId int
	err = connection.QueryRow(
		context.Background(),
		"INSERT INTO todos (completed, body) VALUES ($1, $2) RETURNING id",
		todo.Completed,
		todo.Body,
	).Scan(&insertedId)

	if err != nil {
		log.Fatal("Error inserting todo", err)
	}

	todo.ID = insertedId

	return c.Status(201).JSON(todo)
}

func updateTodo(c *fiber.Ctx, connection *pgx.Conn) error {
	// Retrieve the ID from the URL parameter
	id := c.Params("id")

	// Check the current status of the todo's completed field
	var currentCompleted bool
	err := connection.QueryRow(context.Background(), "SELECT completed FROM todos WHERE id = $1", id).Scan(&currentCompleted)
	if err != nil {
		// If no rows are found (todo doesn't exist), return a 404 error
		if err.Error() == "no rows in result set" {
			return c.Status(404).JSON(fiber.Map{
				"error": "Todo not found",
			})
		}
		// If there is another error, log and return a 500 error
		log.Fatal("Error querying todo status", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Toggle the completed status
	newCompleted := !currentCompleted

	// Update the todo's completed status in the database
	updateQuery := `UPDATE todos SET completed = $1 WHERE id = $2 RETURNING id, completed`
	row := connection.QueryRow(context.Background(), updateQuery, newCompleted, id)

	// Store the updated Todo into the todo struct
	var updatedTodo Todo
	err = row.Scan(&updatedTodo.ID, &updatedTodo.Completed)
	if err != nil {
		log.Fatal("Error updating todo", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Return the updated todo
	return c.JSON(updatedTodo)
}

func deleteTodo(c *fiber.Ctx, connection *pgx.Conn) error {
	id := c.Params("id")

	deleteQuery := `DELETE FROM todos WHERE id = $1`
	result, err := connection.Exec(context.Background(), deleteQuery, id)
	if err != nil {
		log.Fatal("Error deleting todo", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})

	}

	// Check if any rows were affected (meaning the todo was deleted)
	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Todo not found",
		})
	}

	return c.Status(200).JSON(fiber.Map{"success": true})
}
