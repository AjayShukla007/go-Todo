package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt/options"
)

type Todo struct {
	ID    primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title string             `json:"title"`
	Done  bool               `json:"done"`
}

var collection *mongo.Collection

func main() {
	fmt.Println("Hello World")
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("error: Error loading .env file,", err)
	}
	MONGO_URI := os.Getenv("MONGO_URI")
	clientOption := options.Client().ApplyURI(MONGO_URI)
	client, err := mongo.Connect(context.Background(), clientOption)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected to mongo")

	collection = client.Database("golang_db").Collection("todos")

	app := fiber.New()

	app.Get("/", getHandler)
	app.Post("/api/post", postHandler)
	app.Patch("/api/updateTodo/:id", updateHandler)
	app.Delete("/api/deleteTodo/:id", delHandler)

	port := os.Getenv("PORT")

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
		return err
	}
	if todo.Title == "" {
		return c.Status(400).JSON(fiber.Map{"error": "title cannot be empty string"})
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
