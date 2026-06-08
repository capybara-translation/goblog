package goblog

import (
	"github.com/capybara-translation/goblog/internal/db"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/jmoiron/sqlx"
)

// InitSchema runs all migrations and then seeds essential default data
// (the built-in reaction types). Every entrypoint (server, seed, adduser)
// calls this so a fresh DB always reaches the same fully-seeded state, and
// the default reaction emojis come from a single Go source of truth.
func InitSchema(database *sqlx.DB) error {
	if err := db.RunMigrations(database, Migrations, MigrationFiles...); err != nil {
		return err
	}
	return repo.SeedReactionTypes(database)
}
