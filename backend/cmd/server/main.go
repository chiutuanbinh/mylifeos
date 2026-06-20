package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/chiutuanbinh/mylifeos/backend/internal/infra/postgres"
	infraevents "github.com/chiutuanbinh/mylifeos/backend/internal/infra/events"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/migrate"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
	"github.com/chiutuanbinh/mylifeos/backend/internal/scraper"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
)

func main() {
	_ = godotenv.Load("../../.env.local")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := postgres.NewPool(context.Background())
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

	// Repos
	txRepo       := postgres.NewTransactionRepo(db)
	krLogRepo    := postgres.NewKRLogRepo(db)
	goalRepo     := postgres.NewGoalRepo(db)
	noteRepo     := postgres.NewNoteRepo(db)
	eventRepo    := postgres.NewEventRepo(db)
	assetRepo    := postgres.NewAssetRepo(db)
	liabRepo     := postgres.NewLiabilityRepo(db)
	settingsRepo := postgres.NewSettingsRepo(db)
	trendsRepo   := postgres.NewTrendsRepo(db)

	// scraperRepo uses the old repo interface expected by scraper.Run
	scraperRepo := repo.NewTrendsRepo(db)

	// Accounting repos, services, handlers
	accountRepo     := postgres.NewAccountRepo(db)
	journalRepo     := postgres.NewJournalRepo(db)
	eventPub        := infraevents.NewInProcessPublisher()
	accountSvc      := accountingsvc.NewAccountService(accountRepo)
	journalSvc      := accountingsvc.NewJournalService(journalRepo, accountRepo, eventPub)
	nwQuery         := accountingsvc.NewNetWorthQuery(accountRepo, journalRepo)
	accountsHandler := httphandler.NewAccountsHandler(accountSvc, journalRepo)
	journalHandler  := httphandler.NewJournalHandler(journalSvc, nwQuery)

	// Services
	dashSvc := dashboardsvc.New(assetRepo, liabRepo, txRepo, goalRepo, trendsRepo)

	// Handlers
	dashHandler    := httphandler.NewDashboardHandler(dashSvc)
	txHandler      := httphandler.NewTransactionHandler(txRepo)
	krLogHandler   := httphandler.NewKRLogHandler(krLogRepo)
	goalHandler    := httphandler.NewGoalHandler(goalRepo)
	noteHandler    := httphandler.NewNoteHandler(noteRepo)
	eventHandler   := httphandler.NewEventHandler(eventRepo)
	gcalHandler    := httphandler.NewGoogleCalendarHandler(eventRepo)
	assetHandler   := httphandler.NewAssetHandler(assetRepo)
	liabHandler    := httphandler.NewLiabilityHandler(liabRepo)
	settingHandler := httphandler.NewSettingsHandler(settingsRepo)
	trendsHandler  := httphandler.NewTrendsHandler(trendsRepo, assetRepo, scraperRepo)

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

		r.Get("/liabilities",         liabHandler.List)
		r.Post("/liabilities",         liabHandler.Create)
		r.Patch("/liabilities/{id}",   liabHandler.Update)
		r.Delete("/liabilities/{id}",  liabHandler.Delete)

		r.Get("/settings",  settingHandler.Get)
		r.Put("/settings",  settingHandler.Update)

		r.Get("/net-worth-snapshots",  trendsHandler.ListSnapshots)
		r.Post("/net-worth-snapshots", trendsHandler.AddSnapshot)
		r.Get("/benchmarks",           trendsHandler.ListBenchmarks)
		r.Get("/bank-rates",           trendsHandler.ListBankRates)
		r.Get("/news",                 trendsHandler.ListNews)
		r.Post("/scrape",              trendsHandler.TriggerScrape)

		r.Get("/accounts",           accountsHandler.List)
		r.Post("/accounts",          accountsHandler.Create)
		r.Post("/journal/entries",   journalHandler.RecordTransaction)
		r.Get("/journal/networth",   journalHandler.NetWorth)
	})

	go func() {
		scraper.Run(context.Background(), scraperRepo)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			scraper.Run(context.Background(), scraperRepo)
		}
	}()

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
