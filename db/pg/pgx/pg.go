package pgx

import (
	"context"
	"io"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/pkg/errors"
)

// DB extends pgxpool.Pool functionality
type DB struct {
	*pgxpool.Pool
	io.Closer
}

type Options struct {
	Tracers []pgx.QueryTracer
}

func New(cfg Config, options *Options) (*DB, error) {
	dsn := cfg.URL().String()
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pgxpool.ParseConfig")
	}

	if cfg.MaxOpenConns < 1 {
		cfg.MaxOpenConns = 1
	}
	poolCfg.MaxConns = cfg.MaxOpenConns
	poolCfg.MaxConnLifetime = time.Duration(cfg.MaxConnLifeTime) * time.Second
	poolCfg.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Second
	poolCfg.HealthCheckPeriod = 20 * time.Second

	if options == nil {
		options = &Options{}
	}

	if len(options.Tracers) > 0 {
		poolCfg.ConnConfig.Tracer = multitracer.New(options.Tracers...)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init database connections pool")
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return &DB{Pool: pool}, nil
}

func NewDefault(c Config) (*DB, error) {
	return New(c, &Options{
		Tracers: []pgx.QueryTracer{
			otelpgx.NewTracer(),
			&tracelog.TraceLog{
				Logger:   NewLogger(),
				LogLevel: parseTraceLogLevel(c.TraceLogLevel),
			},
		},
	})
}

func (db *DB) Close() error {
	db.Pool.Close()
	return nil
}
