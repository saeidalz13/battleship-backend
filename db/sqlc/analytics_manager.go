package sqlc

import (
	"context"

	"github.com/sqlc-dev/pqtype"
)

type AnalyticsManager struct {
	queries Querier
}

func NewAnalyticsManager(queries Querier) *AnalyticsManager {
	return &AnalyticsManager{queries: queries}
}

func (a *AnalyticsManager) IncrementGamesCreatedCount(ctx context.Context, serverIpNet pqtype.Inet) error {
	return a.queries.IncrementGamesCreatedCount(ctx, serverIpNet)
}

func (a *AnalyticsManager) IncrementRematchCalledCount(ctx context.Context, serverIpNet pqtype.Inet) error {
	return a.queries.IncrementRematchCalledCount(ctx, serverIpNet)
}

func (a *AnalyticsManager) GetGamesCreatedCount(ctx context.Context, serverIpNet pqtype.Inet) (int64, error) {
	return a.queries.GetGamesCreatedCount(ctx, serverIpNet)
}

func (a *AnalyticsManager) GetRematchCalledCount(ctx context.Context, serverIpNet pqtype.Inet) (int64, error) {
	return a.queries.GetRematchCalledCount(ctx, serverIpNet)
}
