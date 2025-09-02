package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"

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
	Save(ctx context.Context, se types.Event, device dmc.Device, payload types.SensorPayload, objects []lwm2m.Lwm2mObject, err error) error
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

func (s *postgres) Save(ctx context.Context, se types.Event, device dmc.Device, payload types.SensorPayload, objects []lwm2m.Lwm2mObject, err error) error {
	log := logging.GetFromContext(ctx)

	evt, err := json.Marshal(se)
	if err != nil {
		log.Error("could not marshal sensor event", "err", err.Error())
		return err
	}

	args := pgx.NamedArgs{
		"sensor_id": se.DevEUI,
		"device_id": nil,
		"event":     evt,
		"payload":   nil,
		"objects":   nil,
		"error":     nil,
		"trace_id":  nil,
	}

	if device != nil {
		args["device_id"] = device.ID()
	}

	if payload != nil {
		p, _ := json.Marshal(payload)
		args["payload"] = p
	}

	if len(objects) > 0 {
		o, _ := json.Marshal(objects)
		args["objects"] = o
	}

	if err != nil {
		args["error"] = err.Error()
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID()
		args["trace_id"] = traceID.String()
	}

	sql := `INSERT INTO sensor_events_v2 (sensor_id, device_id, event, payload, objects, error, trace_id) VALUES (@sensor_id, @device_id, @event, @payload, @objects, @error, @trace_id);`

	_, err = s.conn.Exec(ctx, sql, args)
	if err != nil {
		log.Error("could not save sensor event", "sql", sql, "args", args, "err", err.Error())
		return err
	}

	return nil
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

		CREATE TABLE IF NOT EXISTS sensor_events_v2 (
			id 			UUID DEFAULT gen_random_uuid(),
			time 		TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			sensor_id   TEXT NOT NULL,
			device_id   TEXT NULL,
			event       JSONB NULL,
			payload     JSONB NULL,
			objects     JSONB NULL,
			error       TEXT NULL,
			trace_id 	TEXT NULL,
			PRIMARY KEY (time, id)
		);

		DO $$
		DECLARE
			n INTEGER;
		BEGIN
			SELECT COUNT(*) INTO n
			FROM timescaledb_information.hypertables
			WHERE hypertable_name = 'sensor_events_v2';

			IF n = 0 THEN
				PERFORM create_hypertable('sensor_events_v2', 'time');
			END IF;
		END $$;

		DROP TABLE IF EXISTS agent_sensor_events;
		DROP TABLE IF EXISTS sensor_events;
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
