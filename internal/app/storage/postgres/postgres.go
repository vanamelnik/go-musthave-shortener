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

var _ storage.Storage = (*Repo)(nil)

type Repo struct {
	db *sql.DB
}

func NewRepo(dsn string) (*Repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	r := Repo{db: db}
	err = r.AutoMigrate()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (r Repo) AutoMigrate() error {
	const query = `CREATE TABLE IF NOT EXISTS repo (id TEXT, key TEXT UNIQUE, url TEXT UNIQUE);`
	_, err := r.db.ExecContext(context.Background(), query) // мы не используем передачу контекста, поскольку пока не планируется механизма завершения транзакций извне по какому-либо событию

	return err
}

func (r Repo) DestructiveReset() error {
	const query = `DROP TABLE IF EXISTS repo;`
	res, err := r.db.ExecContext(context.Background(), query)
	if err != nil {
		return err
	}
	log.Printf("postgres: drop table: %+v", res)

	return r.AutoMigrate()
}

func (r Repo) Store(id uuid.UUID, key, url string) error {
	ctx := context.Background()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO repo (id, key, url) VALUES ($1,$2,$3)
		ON CONFLICT (url) DO NOTHING;`,
		id.String(), key, url)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		row := r.db.QueryRowContext(ctx, "SELECT key FROM repo WHERE url=$1", url)
		if err = row.Scan(&key); err != nil {
			return err
		}
		return &storage.ErrURLArlreadyExists{ // возвращаем имеющиеся ключ с URL'ом в теле ошибки.
			Key: key,
			URL: url,
		}
	}

	return nil
}

func (r Repo) GetAll(id uuid.UUID) map[string]string {
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

func (r Repo) Get(key string) (string, error) {
	row := r.db.QueryRowContext(context.Background(),
		`SELECT url FROM repo WHERE key=$1;`, key)
	var url string
	err := row.Scan(&url)

	return url, err
}

func (r Repo) Close() {
	r.db.Close()
	log.Println("postgres: database closed")
}

func (r Repo) Ping() error {
	return r.db.Ping()
}

// BatchStore - реализация метода интерфейса storage.Storage
func (r Repo) BatchStore(id uuid.UUID, records []storage.Record) error {
	ctx := context.Background()
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	// nolint:errcheck
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO repo (id, key, url) VALUES ($1, $2, $3);")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range records {
		if _, err = stmt.ExecContext(ctx, id, rec.Key, rec.OriginalURL); err != nil {
			return err
		}
	}

	return tx.Commit()
}
