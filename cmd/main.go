package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adrianrios/lunar-test/internal/api"
	"github.com/adrianrios/lunar-test/internal/rockets"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mongoClient := getMongo()
	rocketsAPI := setupRocketsAPI(mongoClient)
	h := api.HandlerFromMux(rocketsAPI, r)

	// HTTP Server
	srv := &http.Server{
		Addr:         ":8088",
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := mongoClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	} else {
		log.Println("Disconnected from MongoDB")
	}

	log.Println("Server stopped")
}

func setupRocketsAPI(mongoClient *mongo.Client) *api.RocketsAPI {
	db := mongoClient.Database(os.Getenv("MONGO_DATABASE"))
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	messagesService := rockets.NewResequencerMessageService(messagesRepository, rocketsRepository)
	rocketsService := rockets.NewRocketsServiceImpl(rocketsRepository)

	return api.NewRocketsAPI(messagesService, rocketsService)
}

func getMongo() *mongo.Client {
	mongoURI := os.Getenv("MONGO_URI")

	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := mongoClient.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")
	return mongoClient
}
