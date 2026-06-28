package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/OSShip/listings/internal/config"
	"github.com/OSShip/listings/internal/events"
	"github.com/OSShip/listings/internal/handler"
	"github.com/OSShip/listings/internal/store"
	"github.com/OSShip/utils/observability"
)

func main() {
	cfg := config.Load()
	observability.InitSentry("listings")
	defer observability.FlushSentry(2 * time.Second)
	logger := observability.InitLogger("listings")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("database connected")

	pub := events.New(cfg.KafkaBrokers)
	defer pub.Close()
	logger.Info("kafka publisher ready", "brokers", cfg.KafkaBrokers)

	h := &handler.Handler{
		Store:  store.New(pool),
		Events: pub,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.SentryHTTPMiddleware)
	r.Use(observability.SentryRecoverMiddleware("listings"))
	r.Use(observability.SentryErrorMiddleware("listings"))
	r.Use(observability.RequestLogMiddleware("listings"))
	r.Use(observability.PrometheusMiddleware("listings"))

	r.Get("/health", observability.HealthHandler("listings"))
	r.Get("/metrics", observability.MetricsHandler().ServeHTTP)

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/", h.Create)
	r.Patch("/{id}", h.Update)

	logger.Info("listings listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
