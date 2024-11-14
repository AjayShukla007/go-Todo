package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func main() {
	fmt.Println("Hello World")

	app := fiber.New()
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("error: loading .env file")
	}
	PORT := os.Getenv("PORT")

	todos := []Todo{}
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{"msg": "hello server", "todos": todos})
	})

	app.Post("api/post", func(c *fiber.Ctx) error {
		todo := &Todo{}
		err := c.BodyParser(todo)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "error saving the todo", "error": err})
		}
		if todo.Title == "" {
			return c.Status(400).JSON(fiber.Map{"message": "Title cannot be empty"})
		}
		todo.ID = len(todos) + 1
		todos = append(todos, *todo)

		return c.Status(201).JSON(todo)
	})

	// update
	app.Patch("api/updateTodo/:id", func (c *fiber.Ctx) error  {
		id := c.Params("id")
		// check := c.Query("check")
		for i, todo := range todos {
			if fmt.Sprint(todo.ID) == id {
				todos[i].Done = true
				return c.Status(200).JSON(todos[i])
			}
		}
		return c.Status(404).JSON(fiber.Map{"error": "todo not found"})
	})
	// delete
	app.Patch("api/deleteTodo/:id", func (c *fiber.Ctx) error  {
		id := c.Params("id")
		// check := c.Query("check")
		for i, todo := range todos {
			if fmt.Sprint(todo.ID) == id {
				todos = append(todos[:i], todos[i+1:]...)
				return c.Status(200).JSON(fiber.Map{"success": true})
			}
		}
		return c.Status(404).JSON(fiber.Map{"error": "todo not found"})
	})
	log.Fatal(app.Listen(":"+PORT))

}

/*
type employee interface {
	getName() string
			getSalary() int
		}

		type contractor struct {
			name string
			hourlyRate int
			hoursWorked int
		}

		func (e contractor) getName() string {
			return e.name
		}

		func (e contractor) getSalary() int {
			return e.hourlyRate * e.hoursWorked
		}

		type fullTime struct {
			name string
			monthlySalary int
		}

		func (e fullTime) getName() string {
			return e.name
		}

		func (e fullTime) getSalary() int {
			return e.monthlySalary
		}

		type training struct {
			name string
			hourlyRate int
			hoursWorked int
		}

		func (e training) getName() string {
			return e.name
		}

		func (e training) getSalary() int {
			total := e.hourlyRate * e.hoursWorked
			training := total * 12 / 100
			return total - training
		}

		func test(e employee) {
			fmt.Println(e.getName(), e.getSalary())
		}
*/
/* test(contractor{"Ajay", 100, 10})
test(fullTime{"Vijay", 1200})
test(training{"Raj", 100, 10}) */
