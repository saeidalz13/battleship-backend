package sqlc

import "time"

const (
	QuerierCtxTimeout = time.Second * 10
)

type DbManager struct {
	Analytics *AnalyticsManager
}

func NewDbManager(queries Querier) DbManager {
	return DbManager{
		Analytics: NewAnalyticsManager(queries),
	}
}
