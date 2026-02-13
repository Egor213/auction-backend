package postgres

import (
	"context"
	"time"

	errutils "auction-platform/pkg/errors"

	"github.com/Masterminds/squirrel"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultMaxPoolSize  = 1
	DefaultConnAttempts = 10
	DefaultConnTimeout  = time.Second
)

type Postgres struct {
	Builder      squirrel.StatementBuilderType
	CtxGetter    *trmpgx.CtxGetter
	Pool         *pgxpool.Pool
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration
}

func New(pgUrl string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  DefaultMaxPoolSize,
		connAttempts: DefaultConnAttempts,
		connTimeout:  DefaultConnTimeout,
		CtxGetter:    trmpgx.DefaultCtxGetter,
		Builder:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(pgUrl)
	if err != nil {
		return nil, errutils.WrapPathErr(err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize)

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			return nil, errutils.WrapPathErr(err)
		}

		if err = pg.Pool.Ping(context.Background()); err == nil {
			break
		}

		pg.connAttempts--
		log.Infof("Postgres trying to connect, attempts left: %d", pg.connAttempts)
		time.Sleep(pg.connTimeout)
	}

	if err != nil {
		return nil, errutils.WrapPathErr(err)
	}

	return pg, nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
