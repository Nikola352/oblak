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
	f := Function{
		FunctionId: uuid.New(),
		UserId:     userId,
		Status:     status,
		Bucket:     bucket,
		Path:       path,
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO functions (function_id, user_id, status, bucket, path)
		VALUES ($1, $2, $3, $4, $5)
	`, f.FunctionId, f.UserId, f.Status, f.Bucket, f.Path)
	if err != nil {
		return Function{}, fmt.Errorf("create function: %w", err)
	}
	return f, nil
}

func (s *Store) UpdateFunctionStatus(ctx context.Context, functionId uuid.UUID, status Status) error {
	_, err := s.db.Exec(ctx, `
		UPDATE functions SET status = $1 WHERE function_id = $2
	`, status, functionId)
	if err != nil {
		return fmt.Errorf("update status function: %w", err)
	}
	return nil
}

func (s *Store) UpdateFunctionStatusIf(ctx context.Context, functionId uuid.UUID, from, to Status) (bool, error) {
	result, err := s.db.Exec(ctx, `
          UPDATE functions 
          SET status = $1 
          WHERE function_id = $2 AND status = $3
      `, to, functionId, from)
	if err != nil {
		return false, fmt.Errorf("update status function: %w", err)
	}
	return result.RowsAffected() == 1, nil
}
