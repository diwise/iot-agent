package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/farshidtz/senml/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

//go:generate moq -rm -out storage_mock.go . Storage

type Storage interface {
	Initialize(context.Context) error
	Add(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error
	GetMeasurements(ctx context.Context, id, temprel string, t, et time.Time, lastN int) ([]senml.Pack, error)
}

type impl struct {
	db *pgxpool.Pool
}

type Config struct {
	host     string
	user     string
	password string
	port     string
	dbname   string
	sslmode  string
}

func LoadConfiguration(log zerolog.Logger) Config {
	return Config{
		host:     env.GetVariableOrDefault(log, "POSTGRES_HOST", ""),
		user:     env.GetVariableOrDefault(log, "POSTGRES_USER", ""),
		password: env.GetVariableOrDefault(log, "POSTGRES_PASSWORD", ""),
		port:     env.GetVariableOrDefault(log, "POSTGRES_PORT", "5432"),
		dbname:   env.GetVariableOrDefault(log, "POSTGRES_DBNAME", "diwise"),
		sslmode:  env.GetVariableOrDefault(log, "POSTGRES_SSLMODE", "disable"),
	}
}

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

func Connect(ctx context.Context, log zerolog.Logger, cfg Config) (Storage, error) {
	conn, err := pgxpool.New(ctx, cfg.ConnStr())
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("connected to %s...", cfg.host)

	return &impl{
		db: conn,
	}, nil
}

func (i *impl) Initialize(ctx context.Context) error {
	return i.createTables(ctx)
}

func (i *impl) createTables(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS measurements (
			time 		TIMESTAMPTZ NOT NULL,
			corr_id		UUID NOT NULL,			
			device_id 	TEXT NOT NULL,
			PRIMARY KEY (corr_id)
		);

		CREATE INDEX IF NOT EXISTS measurements_device_id_idx ON measurements (device_id);

		CREATE TABLE IF NOT EXISTS measurements_values (
			time 		TIMESTAMPTZ NOT NULL,
			corr_id		UUID NOT NULL,
			row_id		SMALLINT NOT NULL,
			bn 			TEXT NOT NULL DEFAULT '',
			bt 			NUMERIC (13,0) NOT NULL DEFAULT 0,
			bu 			TEXT NOT NULL DEFAULT '',
			bver 		INTEGER NULL,
			bv 			NUMERIC NULL,
			bs 			NUMERIC NULL,
			n 			TEXT NOT NULL DEFAULT '',
			u 			TEXT NOT NULL DEFAULT '',
			t 			NUMERIC (13,0) NOT NULL DEFAULT 0,
			ut 			NUMERIC (13,0) NOT NULL DEFAULT 0,
			v 			NUMERIC NULL,
			vs 			TEXT NOT NULL DEFAULT '',
			vd 			TEXT NOT NULL DEFAULT '',
			vb 			BOOLEAN NULL,
			s 			NUMERIC NULL,
			UNIQUE ("time", corr_id, row_id),
			CONSTRAINT fk_measurement
      			FOREIGN KEY(corr_id) 
	  				REFERENCES measurements(corr_id)
	  				ON DELETE CASCADE
	  	);`

	tx, err := i.db.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, ddl)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	var n int32
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) n
		FROM timescaledb_information.hypertables
		WHERE hypertable_name = 'measurements_values';`).Scan(&n)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if n == 0 {
		_, err := tx.Exec(ctx, `SELECT create_hypertable('measurements_values', 'time');`)
		if err != nil {
			tx.Rollback(ctx)
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (i *impl) Add(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error {
	insert := `INSERT INTO measurements("time", corr_id, device_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`

	insertValue := `INSERT INTO measurements_values("time", corr_id, row_id, bn, bt, bu, bver, bv, bs, n, u, t, ut, v, vs, vd, vb, s)
			   		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);`

	err := pack.Validate()
	if err != nil {
		return err
	}

	//pack.Normalize()

	tx, err := i.db.Begin(ctx)
	if err != nil {
		return err
	}

	corrId := uuid.New()

	_, err = tx.Exec(ctx, insert, timestamp, corrId, id)
	if err != nil {
		return err
	}

	errs := make([]error, 0)
	for i, r := range pack {
		// TODO: get timestamp from record
		_, err = tx.Exec(ctx, insertValue, timestamp, corrId, i, r.BaseName, r.BaseTime, r.BaseUnit, r.BaseVersion, r.BaseValue, r.BaseSum, r.Name, r.Unit, r.Time, r.UpdateTime, r.Value, r.StringValue, r.DataValue, r.BoolValue, r.Sum)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		tx.Rollback(ctx)
		return errors.Join(errs...)
	}

	return tx.Commit(ctx)
}

func (i *impl) GetMeasurements(ctx context.Context, id, temprel string, t, et time.Time, lastN int) ([]senml.Pack, error) {
	rows, err := i.db.Query(ctx, `
		SELECT m.corr_id, bn, bt, bu, bver, bv, bs, n, u, t, ut, v, vs, vd, vb, s
		FROM measurements m
		LEFT JOIN measurements_values mv ON m.corr_id = mv.corr_id
		WHERE m.corr_id IN (
			SELECT corr_id 
			FROM measurements 
			WHERE device_id = $1 AND "time" BETWEEN $2 AND $3
			ORDER BY "time" DESC
			LIMIT $4
		)
		GROUP BY m.corr_id, bn, bt, bu, bver, bv, bs, n, u, t, ut, v, vs, vd, vb, s, row_id
		ORDER BY m."time" ASC, mv.row_id ASC`, id, t, et, lastN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packs := make(map[uuid.UUID]senml.Pack)
	errs := make([]error, 0)

	for rows.Next() {
		var corrId uuid.UUID
		var bn, bu, n, u, vs, vd string
		var bt, t, ut float64
		var bv, bs, v, s *float64
		var bver *int
		var vb *bool

		err := rows.Scan(&corrId, &bn, &bt, &bu, &bver, &bv, &bs, &n, &u, &t, &ut, &v, &vs, &vd, &vb, &s)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		r := senml.Record{
			BaseName:    bn,
			BaseTime:    bt,
			BaseValue:   bv,
			BaseUnit:    bu,
			BaseVersion: bver,
			BaseSum:     bs,
			Name:        n,
			Unit:        u,
			Time:        t,
			UpdateTime:  ut,
			Value:       v,
			StringValue: vs,
			DataValue:   vd,
			BoolValue:   vb,
			Sum:         s,
		}

		packs[corrId] = append(packs[corrId], r)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return values(packs), errors.Join(errs...)
}

func values(m map[uuid.UUID]senml.Pack) []senml.Pack {
	p := make([]senml.Pack, len(m))
	i := 0
	for _, v := range m {
		p[i] = v
		i++
	}
	return p
}
