package authkey

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"oblak/internal/server/apperr"

	"github.com/google/uuid"
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

func (s *Store) CreateAuthKey(ctx context.Context, userId uuid.UUID) (AuthKey, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return AuthKey{}, fmt.Errorf("generate secret key: %w", err)
	}
	plaintext := hex.EncodeToString(raw)

	encrypted, err := encrypt(s.kek, plaintext)
	if err != nil {
		return AuthKey{}, fmt.Errorf("encrypt secret key: %w", err)
	}

	authID := uuid.New().String()
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_keys (auth_id, user_id, secret_key)
		VALUES ($1, $2, $3)
	`, authID, userId, encrypted)
	if err != nil {
		return AuthKey{}, fmt.Errorf("create auth key: %w", err)
	}

	return AuthKey{AuthId: authID, UserId: userId, SecretKey: plaintext}, nil
}
