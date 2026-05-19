package function

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) CreateFunction(ctx context.Context, userId uuid.UUID, status Status, bucket, path string) (Function, error) {
	u := Function{
		FunctionId: uuid.New(),
		UserId:     userId,
		Status:     status,
		Bucket:     bucket,
		Path:       path,
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO functions (function_id, user_id, status, bucket, path)
		VALUES ($1, $2, $3, $4, $5)
	`, u.FunctionId, u.UserId, u.Status, u.Bucket, u.Path)
	if err != nil {
		return Function{}, fmt.Errorf("create function: %w", err)
	}
	return u, nil
}
