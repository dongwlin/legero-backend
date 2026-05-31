package migrations

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"
)

//go:embed *.sql
var migrations embed.FS

func Migrate(dsn string) error {
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Error().
			Err(err).
			Msg("failed to run migrations")
		return err
	}

	log.Info().
		Msg("database migrations applied successfully")

	return nil
}
