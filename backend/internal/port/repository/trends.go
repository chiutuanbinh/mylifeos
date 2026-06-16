package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
)

type TrendsRepo interface {
	ListSnapshots(ctx context.Context, userID string) ([]trends.NetWorthSnapshot, error)
	UpsertSnapshot(ctx context.Context, s trends.NetWorthSnapshot) (trends.NetWorthSnapshot, error)
	UpsertBenchmark(ctx context.Context, b trends.BenchmarkData) error
	ListBenchmarks(ctx context.Context, sources []string, from, to string) ([]trends.BenchmarkData, error)
	LatestBankRates(ctx context.Context) ([]trends.BankRate, error)
	UpsertBankRate(ctx context.Context, b trends.BankRate) error
	ListNews(ctx context.Context, limit int) ([]trends.NewsItem, error)
	UpsertNews(ctx context.Context, items []trends.NewsItem) error
}
