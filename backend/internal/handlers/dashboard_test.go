package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockDashRepo struct{}

func (m *mockDashRepo) Summary(_ context.Context, _ string) (models.DashboardSummary, error) {
	return models.DashboardSummary{
		HabitsTotal:      5,
		HabitsDoneToday:  3,
		GoalsAvgProgress: 65,
		BudgetTotal:      3400,
		BudgetSpent:      2100,
		NetWorthTrend:    []float64{110000, 115000, 118500, 121000, 125000, 127450},
		RecentTx:         []models.Transaction{},
	}, nil
}

func TestDashboardSummary(t *testing.T) {
	os.Setenv("ENV", "development")
	os.Setenv("DEV_USER_ID", "test-user")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("DEV_USER_ID")

	h := handlers.NewDashboardHandler(&mockDashRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Summary))

	req := httptest.NewRequest("GET", "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result models.DashboardSummary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.HabitsTotal != 5 {
		t.Errorf("expected 5 habits, got %d", result.HabitsTotal)
	}
}
