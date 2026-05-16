package authkey

import (
	"context"
	"errors"
	"fmt"
	"oblak/internal/server/apperr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db  *pgxpool.Pool
	kek string
}

func NewStore(db *pgxpool.Pool, kek string) *Store {
	return &Store{db: db, kek: kek}
}

func (s *Store) GetAuthKey(ctx context.Context, authId string) (AuthKey, error) {
	rows, err := s.db.Query(ctx, `
		SELECT auth_id, user_id, secret_key
		FROM auth_keys
		WHERE auth_id = $1
	`, authId)
	if err != nil {
		return AuthKey{}, fmt.Errorf("get auth key by auth id: %w", err)
	}

	r, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[AuthKey])
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthKey{}, apperr.ErrNotFound
	}
	if err != nil {
		return AuthKey{}, err
	}

	r.SecretKey, err = decrypt(s.kek, r.SecretKey)
	if err != nil {
		return AuthKey{}, fmt.Errorf("decrypt secret key: %w", err)
	}
	return r, nil
}
