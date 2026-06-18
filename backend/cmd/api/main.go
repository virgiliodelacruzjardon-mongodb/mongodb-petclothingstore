package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"petstore/internal/api"
	"petstore/internal/db"
)

func main() {
	ctx := context.Background()

	store, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Printf("warning: ensure indexes: %v", err)
	}

	h := api.New(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      h.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("API listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
