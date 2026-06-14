package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/migrate"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

func main() {
	_ = godotenv.Load("../../.env.local")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := repo.NewPool(context.Background())
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	// SELF_HOSTED=true: run embedded migrations directly (bypasses Supabase CLI).
	// Leave unset when deploying via Supabase (migrations run in CI via supabase db push).
	if os.Getenv("SELF_HOSTED") == "true" {
		if err := migrate.Run(context.Background()); err != nil {
			log.Fatalf("migrations: %v", err)
		}
	}

	dashHandler    := handlers.NewDashboardHandler(repo.NewDashboardRepo(db))
	txHandler      := handlers.NewTransactionHandler(repo.NewTransactionRepo(db))
	krLogHandler   := handlers.NewKRLogHandler(repo.NewKRLogRepo(db))
	goalHandler    := handlers.NewGoalHandler(repo.NewGoalRepo(db))
	noteHandler    := handlers.NewNoteHandler(repo.NewNoteRepo(db))
	eventRepo      := repo.NewEventRepo(db)
	eventHandler   := handlers.NewEventHandler(eventRepo)
	gcalHandler    := handlers.NewGoogleCalendarHandler(eventRepo)
	assetHandler   := handlers.NewAssetHandler(repo.NewAssetRepo(db))
	settingHandler := handlers.NewSettingsHandler(repo.NewSettingsRepo(db))

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", os.Getenv("FRONTEND_URL")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Get("/dashboard/summary", dashHandler.Summary)

		r.Get("/transactions",        txHandler.List)
		r.Post("/transactions",        txHandler.Create)
		r.Delete("/transactions/{id}", txHandler.Delete)
		r.Get("/budgets",              txHandler.ListBudgets)
		r.Put("/budgets/{category}",   txHandler.UpsertBudget)

		r.Get("/kr-logs",                      krLogHandler.GetLogs)
		r.Get("/key-results/{id}/logs",        krLogHandler.GetLogRange)
		r.Post("/key-results/{id}/log",        krLogHandler.ToggleLog)

		r.Get("/goals",                              goalHandler.List)
		r.Post("/goals",                              goalHandler.Create)
		r.Patch("/goals/{id}",                        goalHandler.Update)
		r.Delete("/goals/{id}",                       goalHandler.Delete)
		r.Post("/goals/{id}/key-results",             goalHandler.AddKeyResult)
		r.Patch("/goals/{id}/key-results/{kr_id}",    goalHandler.UpdateKeyResult)
		r.Delete("/goals/{id}/key-results/{kr_id}",   goalHandler.DeleteKeyResult)

		r.Get("/notes",          noteHandler.List)
		r.Post("/notes",          noteHandler.Create)
		r.Patch("/notes/{id}",    noteHandler.Update)
		r.Delete("/notes/{id}",   noteHandler.Delete)

		r.Get("/events",                    eventHandler.List)
		r.Post("/events",                    eventHandler.Create)
		r.Patch("/events/{id}",              eventHandler.Update)
		r.Delete("/events/{id}",             eventHandler.Delete)
		r.Post("/calendar/google/sync",      gcalHandler.Sync)

		r.Get("/assets",          assetHandler.List)
		r.Post("/assets",          assetHandler.Create)
		r.Patch("/assets/{id}",    assetHandler.Update)
		r.Delete("/assets/{id}",   assetHandler.Delete)

		r.Get("/settings",  settingHandler.Get)
		r.Put("/settings",  settingHandler.Update)
	})

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
