// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0

package sqlc

import (
	"context"

	"github.com/sqlc-dev/pqtype"
)

type Querier interface {
	UpdateGameCreated(ctx context.Context, serverIp pqtype.Inet) error
	UpdateRematchCalled(ctx context.Context, serverIp pqtype.Inet) error
}

var _ Querier = (*Queries)(nil)
