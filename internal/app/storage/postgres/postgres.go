package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
)

var _ storage.Storage = (*Repo)(nil)

type Repo struct {
	db *sql.DB
}

func NewRepo(ctx context.Context, dsn string) (*Repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	r := Repo{db: db}
	err = r.createTable(ctx)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// createTable создает таблицу для хранилища, если она отсутствует.
func (r Repo) createTable(ctx context.Context) error {
	const query = `CREATE TABLE IF NOT EXISTS repo (id TEXT, key TEXT UNIQUE, url TEXT UNIQUE, deleted BOOLEAN DEFAULT FALSE);`
	_, err := r.db.ExecContext(ctx, query)

	return err
}

// destructiveReset удаляет таблицу из хранилища и пересоздаёт её заново.
func (r Repo) destructiveReset(ctx context.Context) error { //nolint unused
	const query = `DROP TABLE IF EXISTS repo;`
	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	log.Printf("postgres: drop table: %+v", res)

	return r.createTable(ctx)
}

func (r Repo) Store(ctx context.Context, id uuid.UUID, key, url string) error {
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
			return fmt.Errorf("postgres: url '%s' already exists in the database, but we cannot get the key: %w", url, err)
		}
		return &storage.ErrURLArlreadyExists{ // возвращаем имеющиеся ключ с URL'ом в теле ошибки.
			Key: key,
			URL: url,
		}
	}

	return nil
}

func (r Repo) GetAll(ctx context.Context, id uuid.UUID) map[string]string {
	m := make(map[string]string)
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, url FROM repo WHERE id=$1 AND NOT deleted;`,
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

func (r Repo) Get(ctx context.Context, key string) (string, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT url, deleted FROM repo WHERE key=$1;`, key)
	var url string
	var deleted bool
	err := row.Scan(&url, &deleted)
	if deleted {
		return "", storage.ErrDeleted
	}

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
func (r Repo) BatchStore(ctx context.Context, id uuid.UUID, records []storage.Record) error {
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
			var pgErr pgx.PgError
			// TODO: в связи с тем, что в сервисе реализовано мягкое удаление, у нас есть проблема -
			// невозможно будет повторно сокращать URL'ы, которые были удалены, т.к. они по-прежнему присутствуют в индексе таблицы.
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return storage.ErrBatchURLUniqueViolation
			}
			return err
		}
	}

	return tx.Commit()
}

func (r Repo) BatchDelete(ctx context.Context, id uuid.UUID, keys []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	// nolint:errcheck
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "UPDATE repo SET deleted=TRUE WHERE id=$1 AND key=$2;")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err = stmt.ExecContext(ctx, id, key); err != nil {
			return err
		}
	}
	return tx.Commit()
}
