package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"
	// "go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt/options"
)

// Todo represents a single todo item in the database
// ID: Unique identifier for the todo item
// Title: The todo item's description
// Done: Indicates if the todo is completed
type Todo struct {
	ID    primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title string             `json:"title"`
	Done  bool               `json:"done"`
}

var collection *mongo.Collection

// BlogsAPI - A RESTful API for managing blog posts
// Author: Ajay Shukla
// Version: 1.0.0
// Environment Variables:
// MONGO_URI - MongoDB connection string
// PORT - Server port number

// @title Blog & Todo API
// @version 2.0
// @description A RESTful API for managing blogs and todos with MongoDB
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
// @contact.name API Support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

type APIError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func handleAPIError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(APIError{
		Status:  status,
		Message: message,
		Code:    fmt.Sprintf("ERR_%d", status),
	})
}

func logMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Printf(
			"method=%s path=%s status=%d duration=%s",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			time.Since(start),
		)
		return err
	}
}

func initializeDatabase(uri string) (*mongo.Client, error) {
	clientOption := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return client, nil
}

type Config struct {
	MongoURI string
	Port     string
	Env      string
}

func loadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}
	
	return &Config{
		MongoURI: os.Getenv("MONGO_URI"),
		Port:     os.Getenv("PORT"),
		Env:      os.Getenv("ENV"),
	}, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, err := initializeDatabase(config.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	collection = client.Database("golang_db").Collection("todos")

	app := fiber.New()
	app.Use(logMiddleware())

	app.Get("/", getHandler)
	app.Post("/api/post", postHandler)
	app.Patch("/api/updateTodo/:id", updateHandler)
	app.Delete("/api/deleteTodo/:id", delHandler)

	port := config.Port

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func getHandler(c *fiber.Ctx) error {
	var todos []Todo
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var todo Todo
		if err := cursor.Decode(&todo); err != nil {
			return err
		}
		todos = append(todos, todo)
	}
	return c.JSON(todos)
}

func postHandler(c *fiber.Ctx) error {
	todo := new(Todo)
	if err := c.BodyParser(todo); err != nil {
		return handleAPIError(c, 400, "Invalid request body")
	}
	if err := validateTodo(todo); err != nil {
		return handleAPIError(c, 400, err.Error())
	}
	insertResult, err := collection.InsertOne(context.Background(), todo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "error adding todo"})
	}
	todo.ID = insertResult.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(todo)
}

func updateHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "empty id"})
	}
	filter := bson.M{"_id": objectId}
	update := bson.M{"$set": bson.M{"done": true}}
	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}
	return c.Status(200).JSON(fiber.Map{"success": true})
}

func delHandler(c *fiber.Ctx) error {

	id := c.Params("id")

	objectId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "empty id"})
	}
	filter := bson.M{"_id": objectId}
	_, err = collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return err
	}
	return c.Status(200).JSON(fiber.Map{"succsess": "todo deleted successfully"})
}

func validateTodo(todo *Todo) error {
	if todo.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(todo.Title) > 100 {
		return fmt.Errorf("title must be less than 100 characters")
	}
	return nil
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

// initializeDatabase establishes connection with MongoDB
// Returns an error if connection fails
// Uses MONGO_URI from environment variables

// getHandler retrieves all todos from the database
// Returns JSON array of todos
// HTTP 200: Success
// HTTP 500: Server error

// postHandler creates a new todo item
// Expects JSON body with title
// HTTP 201: Created successfully
// HTTP 400: Invalid input
// HTTP 500: Server error

// updateHandler marks a todo as complete
// Expects todo ID in URL parameter
// HTTP 200: Updated successfully
// HTTP 400: Invalid ID

// delHandler removes a todo item
// Expects todo ID in URL parameter
// HTTP 200: Deleted successfully
// HTTP 400: Invalid ID
