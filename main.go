package main

import (
	"context"
	"log"
	"net/http"
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

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	pub := events.New(cfg.KafkaBrokers)
	defer pub.Close()

	h := &handler.Handler{
		Store:  store.New(pool),
		Events: pub,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(observability.SentryRecoverMiddleware("listings"))
	r.Use(observability.SentryErrorMiddleware("listings"))
	r.Use(middleware.Recoverer)
	r.Use(observability.PrometheusMiddleware("listings"))

	r.Get("/health", observability.HealthHandler("listings"))
	r.Get("/metrics", observability.MetricsHandler().ServeHTTP)

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/", h.Create)
	r.Patch("/{id}", h.Update)

	log.Printf("listings listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
