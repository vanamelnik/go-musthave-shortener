package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
)

var _ storage.Storage = (*repo)(nil)

type repo struct {
	db *sql.DB
}

func NewRepo(dsn string) (*repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	r := repo{db: db}
	err = r.DestructiveReset() //r.AutoMigrate()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (r repo) AutoMigrate() error {
	const query = `CREATE TABLE IF NOT EXISTS repo (id TEXT, key TEXT UNIQUE, url TEXT);`
	_, err := r.db.ExecContext(context.Background(), query)

	return err
}

func (r repo) DestructiveReset() error {
	const query = `DROP TABLE IF EXISTS repo;`
	res, err := r.db.ExecContext(context.Background(), query)
	if err != nil {
		return err
	}
	log.Printf("postgres: drop table: %+v", res)

	return r.AutoMigrate()
}

func (r repo) Store(id uuid.UUID, key, url string) error {
	_, err := r.db.ExecContext(context.Background(),
		`INSERT INTO repo (id, key, url) VALUES ($1,$2,$3);`,
		id.String(), key, url)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	return nil
}

func (r repo) GetAll(id uuid.UUID) map[string]string {
	m := make(map[string]string)
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT key, url FROM repo WHERE id=$1;`,
		id.String())
	if err != nil {
		log.Printf("postgres: %v", err)
		return m
	}
	defer rows.Close()

	for rows.Next() {
		var key, url string
		err = rows.Scan(&key, &url)
		if err != nil {
			log.Printf("postgres: %v", err)
		}
		m[key] = url
	}

	if rows.Err() != nil {
		log.Printf("postgres: %v", err)
	}

	return m

}

func (r repo) Get(key string) (string, error) {
	row := r.db.QueryRowContext(context.Background(),
		`SELECT url FROM repo WHERE key=$1;`, key)
	var url string
	err := row.Scan(&url)

	return url, err
}

func (r repo) Close() {
	r.db.Close()
	log.Println("postgres: database closed")
}

func (r repo) Ping() error {
	return r.db.Ping()
}
