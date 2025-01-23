package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq"
)

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConnections)
	db.SetMaxIdleConns(cfg.db.maxIdelConnections)
	db.SetConnMaxIdleTime(cfg.db.maxIdelTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

type storage struct {
	db *sql.DB
}

func newStorage(db *sql.DB) *storage {
	return &storage{
		db: db,
	}
}

func (s *storage) getUserByEmail(email string) (*user, error) {
	query := `SELECT id, created_at, name, email, password_hash, is_activated, version
			  FROM users
			  where email = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := s.db.QueryRowContext(ctx, query, email)
	var u user
	err := row.Scan(&u.ID, &u.CreateAt, &u.Name, &u.Email, &u.PasswordHash, &u.IsActivated, &u.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, nil
		default:
			return nil, err
		}
	}
	return &u, nil
}

func (s *storage) getUserByID(id int) (*user, error) {
	query := `SELECT id, created_at, name, email, password_hash, is_activated, version
			  FROM users
			  where id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := s.db.QueryRowContext(ctx, query, id)
	var u user
	err := row.Scan(&u.ID, &u.CreateAt, &u.Name, &u.Email, &u.PasswordHash, &u.IsActivated, &u.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, nil
		default:
			return nil, err
		}
	}
	return &u, nil
}

func (s *storage) insertUser(u *user) error {
	query := `INSERT INTO users (name, email, password_hash, is_activated)
			  VALUES ($1, $2, $3, $4)
			  RETURNING id, created_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := s.db.QueryRowContext(ctx, query, u.Name, u.Email, u.PasswordHash, u.IsActivated)
	err := row.Scan(&u.ID, &u.CreateAt, &u.Version)
	return err
}

func (s *storage) updateUser(u *user) error {
	query := `UPDATE users SET name = $1, email = $2, password_hash = $3, is_activated = $4, version = version + 1
			  WHERE id = $5 and version = $6
			  RETURNING version`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := s.db.QueryRowContext(ctx, query, u.Name, u.Email, u.PasswordHash, u.IsActivated, u.ID, u.Version)
	err := row.Scan(&u.Version)
	return err
}

func (s *storage) deleteUser(u *user) error {
	query := `DELETE from users
			  WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, query, u.ID)
	return err
}
