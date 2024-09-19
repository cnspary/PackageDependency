package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectMongoDB initializes and returns a MongoDB client.
func ConnectMongoDB(uri string) *mongo.Client {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri).SetConnectTimeout(10*time.Second))
	if err != nil {
		fmt.Println("MongoDB connection error:", err)
		return nil
	}
	return client
}

// DisconnectMongoDB closes the MongoDB connection.
func DisconnectMongoDB(client *mongo.Client) {
	if err := client.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}
