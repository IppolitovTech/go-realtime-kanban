package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IppolitovTech/go-realtime-kanban/internal/repository/postgres"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
	transporthttp "github.com/IppolitovTech/go-realtime-kanban/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://kanban:kanban@localhost:5432/kanban?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	boardRepo := postgres.NewBoardRepository(pool)
	columnRepo := postgres.NewColumnRepository(pool)
	cardRepo := postgres.NewCardRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	txManager := postgres.NewTxManager(pool)

	boardService := service.NewBoardService(boardRepo, userRepo, txManager)
	columnService := service.NewColumnService(columnRepo, boardRepo, txManager)
	cardService := service.NewCardService(cardRepo, columnRepo, boardRepo, txManager)

	boardHandler := transporthttp.NewBoardHandler(boardService, columnService, cardService)
	columnHandler := transporthttp.NewColumnHandler(columnService)
	cardHandler := transporthttp.NewCardHandler(cardService)

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		// The frontend (Vite dev server) runs on a different origin than
		// the API — see web/README.md. Overridable for other frontend
		// dev ports/deployments via CORS_ALLOWED_ORIGINS.
		AllowedOrigins:   corsAllowedOrigins(),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete},
		AllowedHeaders:   []string{"Content-Type", "X-User-ID"},
		AllowCredentials: true,
	}))
	router.Get("/health", transporthttp.HealthHandler(pool))

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", transporthttp.HealthHandler(pool))

		r.Group(func(r chi.Router) {
			r.Use(transporthttp.DevUserID)

			r.Route("/boards", func(r chi.Router) {
				r.Get("/", boardHandler.List)
				r.Post("/", boardHandler.Create)

				r.Route("/{boardId}", func(r chi.Router) {
					r.Get("/", boardHandler.Get)
					r.Patch("/", boardHandler.Update)
					r.Delete("/", boardHandler.Delete)
					r.Post("/members", boardHandler.InviteMember)
					r.Post("/columns", columnHandler.Create)
				})
			})

			r.Route("/columns/{columnId}", func(r chi.Router) {
				r.Patch("/", columnHandler.Update)
				r.Delete("/", columnHandler.Delete)
				r.Patch("/move", columnHandler.Move)
				r.Post("/cards", cardHandler.Create)
			})

			r.Route("/cards/{cardId}", func(r chi.Router) {
				r.Patch("/", cardHandler.Update)
				r.Delete("/", cardHandler.Delete)
				r.Patch("/move", cardHandler.Move)
			})
		})
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func corsAllowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		return []string{"http://localhost:5173"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, len(parts))
	for i, p := range parts {
		origins[i] = strings.TrimSpace(p)
	}
	return origins
}
