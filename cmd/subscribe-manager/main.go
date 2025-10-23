package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kulibyka/effective-mobile/internal/config"
	domain "github.com/Kulibyka/effective-mobile/internal/domain/subscription"
	"github.com/Kulibyka/effective-mobile/internal/http/handlers/subscriptions"
	"github.com/Kulibyka/effective-mobile/internal/lib/uuid"
	"github.com/Kulibyka/effective-mobile/internal/logger"
	service "github.com/Kulibyka/effective-mobile/internal/services/subscriptions"
	"github.com/Kulibyka/effective-mobile/internal/storage/postgresql"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("starting app", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	db, err := postgresql.New(cfg.PostgreSQL)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Warn("failed to close postgresql connection", slog.Any("error", err))
		}
	}()

	repo := &storageWrapper{Storage: db}
	subscriptionsService := service.New(repo, log)
	handler := subscriptions.New(subscriptionsService, log)

	mux := http.NewServeMux()
	handler.Register(mux)

	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/swagger" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("docs/swagger"))))

	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      mux,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPServer.Timeout)
		defer cancel()

		log.Info("shutting down http server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to shutdown http server", slog.Any("error", err))
		}
	}()

	log.Info("starting http server", slog.String("address", cfg.HTTPServer.Address))

	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", slog.Any("error", err))
		}
	}
}

func setupLogger(env string) *slog.Logger {
	log := logger.New(env)
	log.Debug("logger configured", slog.String("mode", env))

	return log
}

type storageWrapper struct {
	*postgresql.Storage
}

func (s *storageWrapper) CreateSubscription(ctx context.Context, input domain.CreateInput) (domain.Subscription, error) {
	return s.Storage.CreateSubscription(ctx, input)
}

func (s *storageWrapper) GetSubscription(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	return s.Storage.GetSubscription(ctx, id)
}

func (s *storageWrapper) UpdateSubscription(ctx context.Context, id uuid.UUID, input domain.UpdateInput) (domain.Subscription, error) {
	return s.Storage.UpdateSubscription(ctx, id, input)
}

func (s *storageWrapper) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return s.Storage.DeleteSubscription(ctx, id)
}

func (s *storageWrapper) ListSubscriptions(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, error) {
	return s.Storage.ListSubscriptions(ctx, filter)
}
