package controller

import (
	"context"
	"log"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	AvailabilityCollection *mongo.Collection
	UserCollection         *mongo.Collection
	ProviderCollection     *mongo.Collection
	ServiceCollection      *mongo.Collection
	BookingCollection      *mongo.Collection
	Validate               *validator.Validate
)

func Init(availabilityCollection *mongo.Collection, bookingCollection *mongo.Collection, providerCollection *mongo.Collection, serviceCollection *mongo.Collection, userCollection *mongo.Collection) {
	AvailabilityCollection = availabilityCollection
	BookingCollection = bookingCollection
	ProviderCollection = providerCollection
	ServiceCollection = serviceCollection
	UserCollection = userCollection

	Validate = validator.New()
	EnsureUniqueEmail()
	log.Println("Controller initialized successfully.")
}

func EnsureUniqueEmail() {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	indexName, err := UserCollection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Println("Error creating index:", err)
		return
	}
	log.Println("Index created successfully:", indexName)
}
