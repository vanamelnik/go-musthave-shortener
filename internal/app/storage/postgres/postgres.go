package postgres

import (
	"database/sql"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type Repo struct {
	db *sql.DB
}

func NewDB(dsn string) (*Repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return &Repo{db: db}, nil
}

func (r Repo) Close() {
	r.db.Close()
}

func (r Repo) Ping() error {
	return r.db.Ping()
}
