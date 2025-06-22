package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"workflow-code-test/api/pkg/db"
	"workflow-code-test/api/services/workflow"
)

func main() {
	// Configure structured logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(logHandler))

	dbConfig := db.DefaultConfig()
	dbConfig.URI = os.Getenv("DATABASE_URL")

	if err := db.Connect(dbConfig); err != nil {
		slog.Error("Failed to connect to database", "error", err, "dbConfig", dbConfig.URI)
		return
	}

	defer db.Disconnect()

	// Initialise database schema and seed data
	ctx := context.Background()
	if err := db.InitDatabase(ctx, db.GetPool()); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		return
	}

	// setup router
	mainRouter := mux.NewRouter()

	apiRouter := mainRouter.PathPrefix("/api/v1").Subrouter()

	workflowService, err := workflow.NewService(db.GetPool())
	if err != nil {
		slog.Error("Failed to create workflow service", "error", err)
		return
	}

	workflowService.LoadRoutes(apiRouter, false)

	// Configure CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3003"}), // Frontend URL
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		handlers.AllowCredentials(),
	)(mainRouter)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: corsHandler,
	}

	// Channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		slog.Info("Starting server on :8080")
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking select waiting for either a signal or an error
	select {
	case err := <-serverErrors:
		slog.Error("Server error", "error", err)

	case sig := <-shutdown:
		slog.Info("Shutdown signal received", "signal", sig)

		// Give outstanding requests 5 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Could not stop server gracefully", "error", err)
			srv.Close()
		}
	}
}
