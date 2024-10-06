package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
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
	Detail  string `json:"detail"`
}

// APIError provides standardized error response
// Status: HTTP status code
// Message: Human readable error message
// Code: Machine readable error code
// Detail: Additional error context

func handleAPIError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(APIError{
		Status:  status,
		Message: message,
		Code:    fmt.Sprintf("ERR_%d", status),
		Detail:  fmt.Sprintf("Detailed explanation for %d error", status),
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
	maxRetries := 3
	var client *mongo.Client
	var err error

	for i := 0; i < maxRetries; i++ {
		clientOptions := options.Client().ApplyURI(uri)
		client, err = mongo.Connect(context.Background(), clientOptions)
		if err == nil {
			if err = client.Ping(context.Background(), nil); err == nil {
				return client, nil
			}
		}
		log.Printf("Failed to connect to MongoDB (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * 2)
	}
	return nil, fmt.Errorf("failed to connect after %d attempts", maxRetries)
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

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
	Database  string `json:"database"`
}

func healthHandler(c *fiber.Ctx) error {
	// Check database connection status
	// Returns "connected" if ping successful, "disconnected" otherwise
	dbStatus := "connected"
	if err := client.Ping(context.Background(), nil); err != nil {
		dbStatus = "disconnected"
	}
	return c.JSON(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "2.0.0",
		Database:  dbStatus,
	})
}

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
	}
}

func (rl *RateLimiter) Limit(maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		now := time.Now()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		requests := rl.requests[ip]

		// Remove old requests outside the window
		for i, t := range requests {
			if now.Sub(t) > window {
				requests = requests[i+1:]
				break
			}
		}

		if len(requests) >= maxRequests {
			return handleAPIError(c, 429, "Too many requests")
		}

		rl.requests[ip] = append(requests, now)
		return c.Next()
	}
}

func timeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- c.Next()
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return handleAPIError(c, 408, "Request timeout")
		}
	}
}

func corsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		return c.Next()
	}
}

func jwtMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return handleAPIError(c, 401, "No authorization token provided")
		}
		// Token validation logic here
		return c.Next()
	}
}

type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string             `json:"username"`
	Password string             `json:"password"`
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func registerHandler(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return handleAPIError(c, 400, "Invalid user data")
	}
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return handleAPIError(c, 500, "Failed to hash password")
	}
	user.Password = hashedPassword
	// User saving logic here
	return c.Status(201).SendString("User registered successfully")
}

func sendPasswordResetEmail(email string) error {
	// Email sending logic here
	return nil
}

func validateEmail(email string) error {
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func passwordResetHandler(c *fiber.Ctx) error {
	email := c.FormValue("email")
	if email == "" {
		return handleAPIError(c, 400, "Email is required")
	}
	err := validateEmail(email)
	if err != nil {
		return handleAPIError(c, 400, err.Error())
	}
	err = sendPasswordResetEmail(email)
	if err != nil {
		return handleAPIError(c, 500, "Failed to send password reset email")
	}
	return c.SendStatus(200)
}

func generateJWT(user User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.ID.Hex()
	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func loginHandler(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return handleAPIError(c, 400, "Invalid login data")
	}
	// User authentication logic here
	token, err := generateJWT(*user)
	if err != nil {
		return handleAPIError(c, 500, "Failed to generate token")
	}
	return c.JSON(fiber.Map{"token": token})
}

func updateUserHandler(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return handleAPIError(c, 400, "Invalid user data")
	}
	// User update logic here
	return c.SendStatus(200)
}

func userActivityLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("User activity: %s %s", c.Method(), c.Path())
		return c.Next()
	}
}

type Comment struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	TodoID primitive.ObjectID `json:"todoId" bson:"todoId"`
	Text   string             `json:"text"`
}

func postCommentHandler(c *fiber.Ctx) error {
	comment := new(Comment)
	if err := c.BodyParser(comment); err != nil {
		return handleAPIError(c, 400, "Invalid comment data")
	}
	_, err := collection.InsertOne(context.Background(), comment)
	if err != nil {
		return handleAPIError(c, 500, "Failed to add comment")
	}
	return c.Status(201).JSON(comment)
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
	limiter := NewRateLimiter()
	app.Use(limiter.Limit(100, time.Minute))
	app.Use(logMiddleware())
	app.Use(timeoutMiddleware(10 * time.Second))
	app.Use(corsMiddleware())
	app.Use(jwtMiddleware())
	app.Use(userActivityLogger())

	app.Get("/", getHandler)
	app.Post("/api/post", postHandler)
	app.Patch("/api/updateTodo/:id", updateHandler)
	app.Delete("/api/deleteTodo/:id", delHandler)
	app.Get("/health", healthHandler)
	app.Post("/api/register", registerHandler)
	app.Post("/api/passwordReset", passwordResetHandler)
	app.Post("/api/login", loginHandler)
	app.Patch("/api/user/:id", updateUserHandler)
	app.Post("/api/todo/:id/comment", postCommentHandler)

	port := config.Port

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func getHandler(c *fiber.Ctx) error {
	var todos []Todo
	opts := options.Find().SetSort(bson.D{{"created_at", -1}})
	cursor, err := collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return handleAPIError(c, 500, "Database query failed")
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var todo Todo
		if err := cursor.Decode(&todo); err != nil {
			return handleAPIError(c, 500, "Failed to decode todo")
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
	if strings.TrimSpace(todo.Title) == "" {
		return fmt.Errorf("title cannot be only whitespace")
	}
	return nil
}

const (
	ErrInvalidEmail    = "Invalid email format"
	ErrTitleRequired   = "Title is required"
	ErrTitleTooLong    = "Title must be less than 100 characters"
	ErrTitleWhitespace = "Title cannot be only whitespace"
)

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

// RateLimiter implements request rate limiting per IP
// Uses sliding window algorithm for accurate rate tracking

// TODO: Move JWT secret to environment variables
// Current implementation uses hardcoded secret for development
