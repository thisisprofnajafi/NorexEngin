package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func Connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB connection string
	clientOptions := options.Client().ApplyURI("mongodb://services.frn2.chabokan.net:47161").SetAuth(options.Credential{
		Username: "root",             // Replace with your MongoDB username
		Password: "cfEU9De2uPsghDsi", // Replace with your MongoDB password
	})

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}

	Client = client
	log.Println("Connected to MongoDB!")
}

func GetCollection(collectionName string) *mongo.Collection {
	return Client.Database("norex_db").Collection(collectionName)
}
