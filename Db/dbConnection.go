package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB() (*mongo.Client, error) {
	connectionUrl := os.Getenv("MONGO_URI")
	if connectionUrl == "" {
		log.Fatal("MONGO_URI environment variable not set")
	}

	clientOption := options.Client().ApplyURI(connectionUrl)
	client, err := mongo.Connect(context.Background(), clientOption)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}
	fmt.Println("Successfully Connected to DB")
	return client, nil
}
