package main

import (
	"context"
	"log"

	"cryptorg/internal/app"
)

func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	ctx := context.Background()
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
