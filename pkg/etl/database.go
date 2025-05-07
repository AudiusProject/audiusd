package etl

import (
	"context"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ETLDatabase is an interface for writing to the ETL database
// the default implementation is a postgres writer
type ETLDatabase interface {
	WritePlays(ctx context.Context, plays []*v1.TrackPlay) error
}

var _ ETLDatabase = (*PostgresETLWriter)(nil)

type PostgresETLWriter struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresETLWriter(pool *pgxpool.Pool) *PostgresETLWriter {
	return &PostgresETLWriter{
		pool:    pool,
		queries: db.New(pool),
	}
}

// WritePlays implements ETLWriter.
func (p *PostgresETLWriter) WritePlays(ctx context.Context, plays []*v1.TrackPlay) error {
	panic("unimplemented")
}
