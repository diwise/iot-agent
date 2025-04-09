package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	host     string
	user     string
	password string
	port     string
	dbname   string
	sslmode  string
}

func LoadConfiguration(ctx context.Context) Config {
	return Config{
		host:     env.GetVariableOrDefault(ctx, "POSTGRES_HOST", ""),
		user:     env.GetVariableOrDefault(ctx, "POSTGRES_USER", ""),
		password: env.GetVariableOrDefault(ctx, "POSTGRES_PASSWORD", ""),
		port:     env.GetVariableOrDefault(ctx, "POSTGRES_PORT", "5432"),
		dbname:   env.GetVariableOrDefault(ctx, "POSTGRES_DBNAME", "diwise"),
		sslmode:  env.GetVariableOrDefault(ctx, "POSTGRES_SSLMODE", "disable"),
	}
}

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

type Storage interface {
	Save(ctx context.Context, se types.SensorEvent) error
	Close() error
}

type postgres struct {
	conn *pgxpool.Pool
}

func New(ctx context.Context, config Config) (Storage, error) {
	pool, err := connect(ctx, config)
	if err != nil {
		return &postgres{}, err
	}

	err = initialize(ctx, pool)
	if err != nil {
		return &postgres{}, err
	}

	return &postgres{
		conn: pool,
	}, nil
}

func (s *postgres) Close() error {
	if s.conn != nil {
		s.conn.Close()
		return nil
	}
	return nil
}

func (s *postgres) Save(ctx context.Context, se types.SensorEvent) error {
	payload, err := json.Marshal(se)
	if err != nil {
		return err
	}

	sql := `INSERT INTO sensor_events (sensor_id, payload, trace_id) VALUES (@sensor_id, @payload, @trace_id);`

	args := pgx.NamedArgs{
		"sensor_id": se.DevEui,
		"payload":   payload,
		"trace_id":  nil,
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID()
		args["trace_id"] = traceID.String()
	}

	_, err = s.conn.Exec(ctx, sql, args)

	return err
}

func connect(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, config.ConnStr())
	if err != nil {
		return nil, err
	}

	err = p.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func initialize(ctx context.Context, conn *pgxpool.Pool) error {
	ddl := `
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";

		CREATE TABLE IF NOT EXISTS sensor_events (
			id 			UUID DEFAULT gen_random_uuid(),
			time 		TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,			
			sensor_id   TEXT NOT NULL,
			payload     JSONB NULL,			
			trace_id 	TEXT NULL,
			PRIMARY KEY (time, id)
		);

		DO $$
		DECLARE
			n INTEGER;
		BEGIN			
			SELECT COUNT(*) INTO n
			FROM timescaledb_information.hypertables
			WHERE hypertable_name = 'sensor_events';
			
			IF n = 0 THEN				
				PERFORM create_hypertable('sensor_events', 'time');				
			END IF;
		END $$;

		DROP TABLE IF EXISTS agent_sensor_events;
	`

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, ddl)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

type memory struct{}

func (n memory) Save(ctx context.Context, se types.SensorEvent) error {
	return nil
}
func (n memory) Close() error {
	return nil
}

func NewInMemory() (Storage, error) {
	return memory{}, nil
}
