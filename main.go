package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Todo struct {
	ID        bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Completed bool          `json:"completed"`
	Body      string        `json:"body"`
}

var collection *mongo.Collection

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Missing required environment variable %s", key)
	}

	return value
}

func main() {
	// staring the server
	MONGO_DB_URL := getEnv("MONGO_DB_URL")
	clientOptions := options.Client().ApplyURI(MONGO_DB_URL)
	client, err := mongo.Connect(clientOptions)

	if err != nil {
		log.Fatalf("An error occured on mongodb client: %v", err)
	}

	defer client.Disconnect(context.Background())

	// getting the collection
	collection = client.Database("golang").Collection("todos")

	// starting the server
	server := gin.Default()
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://todogolang-frontend.vercel.app/"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "PUT"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://github.com"
		},
		MaxAge: 12 * time.Hour,
	}))
	api := server.Group("/api")

	api.GET("/todos", getTodoes)
	api.POST("/todos", createTodo)
	api.PUT("/todos/:id", updateTodo)
	api.DELETE("/todos", deleteTodoAll)
	api.DELETE("/todos/:id", deleteTodo)

	port := getEnv("PORT")

	server.Run(fmt.Sprintf(":%v", port))
}

func getTodoes(ctx *gin.Context) {
	var todos []Todo

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		returnErr(err, ctx)
		return
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var todo Todo
		if err := cursor.Decode(&todo); err != nil {
			ctx.JSON(400, gin.H{
				"message": err.Error(),
			})
			fmt.Printf("Error: %v\n", err.Error())
			return
		}
		todos = append(todos, todo)
	}

	ctx.JSON(200, todos)
}

func createTodo(ctx *gin.Context) {
	todo := new(Todo)
	err := ctx.BindJSON(&todo)
	if err != nil {
		returnErr(err, ctx)
		return
	}

	if todo.Body == "" {
		ctx.JSON(400, gin.H{
			"message": "body can not be empty",
		})
		fmt.Println("Error: body can not be empty")
		return
	}

	insertResult, err := collection.InsertOne(context.Background(), todo)
	if err != nil {
		returnErr(err, ctx)
		return
	}

	todo.ID = insertResult.InsertedID.(bson.ObjectID)
	ctx.JSON(200, todo)
}

func updateTodo(ctx *gin.Context) {
	id := ctx.Param("id")
	todos, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		returnErr(err, ctx)
		return
	}

	defer todos.Close(context.Background())

	objectId, err := bson.ObjectIDFromHex(id)
	if err != nil {
		returnErr(err, ctx)
		return
	}
	for todos.Next(context.Background()) {
		var todo Todo
		if err := todos.Decode(&todo); err != nil {
			returnErr(err, ctx)
			return
		}

		filer := bson.M{
			"_id": objectId,
		}

		update := bson.M{
			"$set": bson.M{
				"completed": !todo.Completed,
			},
		}

		if todo.ID == objectId {
			_, err := collection.UpdateOne(context.Background(), filer, update)
			if err != nil {
				returnErr(err, ctx)
				return
			}
			ctx.JSON(200, gin.H{
				"message": "succesfully updatedy",
			})
			return
		}
	}

	ctx.JSON(400, gin.H{
		"message": "no entity was found",
	})
}

func deleteTodo(ctx *gin.Context) {
	id := ctx.Param("id")
	objectid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		returnErr(err, ctx)
		return
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{
		"_id": objectid,
	})

	if err != nil {
		returnErr(err, ctx)
		return
	}

	ctx.JSON(200, gin.H{
		"message": "deleted succesfully",
	})
}

func deleteTodoAll(ctx *gin.Context) {
	_, err := collection.DeleteMany(context.Background(), bson.D{})
	if err != nil {
		returnErr(err, ctx)
		return
	}
	ctx.JSON(200, gin.H{
		"message": "deleted succesfully",
	})
}

func returnErr(err error, ctx *gin.Context) {
	ctx.JSON(400, gin.H{
		"message": err.Error(),
	})
	fmt.Println("Error: ", err.Error())
}
