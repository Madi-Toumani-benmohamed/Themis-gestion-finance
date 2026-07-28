package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(url string) (*sql.DB, error) {
	d, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return d, nil
}

// Migrate applique les fichiers SQL embarqués dans l'ordre lexical, chacun
// UNE SEULE FOIS (suivi dans schema_migrations) et en transaction. Sans ce
// suivi, les seeds (INSERT de règles…) seraient réinsérés à chaque démarrage.
func Migrate(d *sql.DB) error {
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		var done bool
		if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, n).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		b, err := migrations.ReadFile("migrations/" + n)
		if err != nil {
			return err
		}
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(b)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", n, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, n); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s (suivi): %w", n, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
