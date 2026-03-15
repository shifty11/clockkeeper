package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	clockkeeper "github.com/rapha/clockkeeper"
	"github.com/rapha/clockkeeper/internal/database"
	"github.com/rapha/clockkeeper/internal/logger"
	"github.com/rapha/clockkeeper/internal/web"
)

// ServeCmd starts the Clock Keeper server.
type ServeCmd struct{}

func (s *ServeCmd) Run() error {
	logger.Setup()
	slog.Info("starting clockkeeper")

	dbConfig := database.LoadConfigFromEnv()
	webConfig := web.LoadConfigFromEnv()

	db, sqlDB, err := database.NewClient(dbConfig)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	defer db.Close()

	slog.Info("database connected")

	staticFiles, err := fs.Sub(clockkeeper.StaticFiles, "web/build")
	if err != nil {
		return err
	}

	server := web.NewServer(webConfig, db, staticFiles)

	go func() {
		if listenErr := server.ListenAndServe(); listenErr != nil && listenErr != http.ErrServerClosed {
			slog.Error("server error", "error", listenErr)
		}
	}()

	slog.Info("clockkeeper is running", "listen", webConfig.Listen)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("clockkeeper stopped")
	return nil
}
