package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

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

func (s Storage) GetSensorEventByID(ctx context.Context, id string) ([]byte, error) {
	row := s.conn.QueryRow(ctx, "SELECT payload FROM sensor_events WHERE id = $1", id)
	var payload json.RawMessage
	err := row.Scan(&payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return payload, nil
}

type sensorEvent struct {
	ID       string          `json:"id"`
	Time     time.Time       `json:"time"`
	SensorID string          `json:"sensor_id"`
	Payload  json.RawMessage `json:"payload"`
	TraceID  *string         `json:"trace_id,omitempty"`
}

func (s Storage) GetSensorEvents(ctx context.Context, params map[string][]string) ([]byte, error) {
	sql := `SELECT id, time, sensor_id, payload, trace_id, count(*) OVER () as total FROM sensor_events WHERE 1=1 `

	args := pgx.NamedArgs{}

	if p, ok := params["sensor_id"]; ok {
		sql += `AND sensor_id = @sensor_id `
		args["sensor_id"] = p[0]
	}

	if p, ok := params["trace_id"]; ok {
		sql += `AND trace_id = @trace_id `
		args["trace_id"] = p[0]
	}

	if p, ok := params["timeAt"]; ok {
		sql += `AND time >= @timeAt `
		args["timeAt"] = p[0]
	}

	if p, ok := params["endTimeAt"]; ok {
		sql += `AND time <= @endTimeAt `
		args["endTimeAt"] = p[0]
	}

	sql += `ORDER BY time ASC `

	var offset, limit int
	var err error

	if p, ok := params["offset"]; ok {
		offset, err = strconv.Atoi(p[0])
		if err == nil {
			sql += `OFFSET @offset `
			args["offset"] = offset
		}
		err = nil
	}

	if p, ok := params["limit"]; ok {
		limit, err = strconv.Atoi(p[0])
		if err == nil {
			sql += `LIMIT @limit `
			args["limit"] = limit
		}
		err = nil
	}

	sql += `; `

	rows, err := s.conn.Query(ctx, sql, args)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	se := make([]sensorEvent, 0)

	var id, sensorID string
	var ts time.Time
	var payload json.RawMessage
	var traceID *string
	var total int64

	for rows.Next() {
		err = rows.Scan(&id, &ts, &sensorID, &payload, &traceID, &total)
		if err != nil {
			return nil, err
		}

		se = append(se, sensorEvent{
			ID:       id,
			Time:     ts,
			SensorID: sensorID,
			Payload:  payload,
			TraceID:  traceID,
		})
	}

	if limit == 0 {
		limit = len(se)
	}

	res := struct {
		Meta struct {
			Total  int64 `json:"total_records"`
			Offset int   `json:"offset"`
			Limit  int   `json:"limit"`
			Count  int   `json:"count"`
		} `json:"meta"`
		Data any `json:"data"`
	}{
		Meta: struct {
			Total  int64 `json:"total_records"`
			Offset int   `json:"offset"`
			Limit  int   `json:"limit"`
			Count  int   `json:"count"`
		}{
			Total:  total,
			Offset: offset,
			Limit:  limit,
			Count:  len(se),
		},
		Data: se,
	}

	return json.Marshal(res)
}

func (s Storage) Close() {
	s.conn.Close()
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
