package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
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

type Storage struct {
	conn *pgxpool.Pool
}

func New(ctx context.Context, config Config) (Storage, error) {
	pool, err := connect(ctx, config)
	if err != nil {
		return Storage{}, err
	}

	err = initialize(ctx, pool)
	if err != nil {
		return Storage{}, err
	}

	return Storage{
		conn: pool,
	}, nil
}

func (s Storage) Save(ctx context.Context, se application.SensorEvent) error {
	payload, err := json.Marshal(se)
	if err != nil {
		return err
	}

	sql := `INSERT INTO agent_sensor_events ("time", sensor_id, payload, trace_id) VALUES (@ts, @sensor_id, @payload, @trace_id);`

	args := pgx.NamedArgs{
		"ts":        se.Timestamp,
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
	createTable := `
			CREATE TABLE IF NOT EXISTS agent_sensor_events (
			time 		TIMESTAMPTZ NOT NULL,
			sensor_id   TEXT NOT NULL,
			payload     JSONB	NULL,
			created_on  timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
			trace_id 	TEXT NULL,
			PRIMARY KEY ("time", sensor_id));
			
			ALTER TABLE agent_sensor_events ADD COLUMN IF NOT EXISTS trace_id TEXT NULL;`

	countHyperTable := `SELECT COUNT(*) n FROM timescaledb_information.hypertables WHERE hypertable_name = 'agent_sensor_events';`

	createHyperTable := `SELECT create_hypertable('agent_sensor_events', 'time');`

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, createTable)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	var n int32
	err = tx.QueryRow(ctx, countHyperTable).Scan(&n)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if n == 0 {
		_, err := tx.Exec(ctx, createHyperTable)
		if err != nil {
			tx.Rollback(ctx)
			return err
		}
	}

	return tx.Commit(ctx)
}
