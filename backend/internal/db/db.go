package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	Client *mongo.Client
	DB     *mongo.Database

	Products   *mongo.Collection
	Categories *mongo.Collection
	Customers  *mongo.Collection
	Orders     *mongo.Collection
	Reviews    *mongo.Collection
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Connect opens a connection to MongoDB 8 and returns a Store with handles to
// every collection used by the app.
func Connect(ctx context.Context) (*Store, error) {
	uri := env("MONGODB_URI", "mongodb://localhost:27017")
	dbName := env("MONGODB_DB", "petstore")

	clientOpts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	// Retry the initial ping so the service survives MongoDB still booting
	// (container start ordering) without relying on compose healthchecks.
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := client.Ping(pingCtx, nil)
		cancel()
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		log.Printf("waiting for MongoDB (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		return nil, lastErr
	}

	d := client.Database(dbName)
	log.Printf("connected to MongoDB database %q", dbName)

	return &Store{
		Client:     client,
		DB:         d,
		Products:   d.Collection("products"),
		Categories: d.Collection("categories"),
		Customers:  d.Collection("customers"),
		Orders:     d.Collection("orders"),
		Reviews:    d.Collection("reviews"),
	}, nil
}
