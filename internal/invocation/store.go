package invocation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) CreateInvocation(ctx context.Context, functionId uuid.UUID, invocationTime time.Time) (Invocation, error) {
	inv := Invocation{
		InvocationId:   uuid.New(),
		FunctionId:     functionId,
		Status:         StatusPending,
		InvocationTime: &invocationTime,
		EndTime:        nil,
		LogPath:        nil,
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO invocations (invocation_id, function_id, status, invocation_time, end_time)
		VALUES ($1, $2, $3, $4, $5)
	`, inv.InvocationId, inv.FunctionId, inv.Status, inv.InvocationTime, inv.EndTime)
	if err != nil {
		return Invocation{}, fmt.Errorf("create invocation: %w", err)
	}
	return inv, nil
}

func (s *Store) UpdateInvocationStatus(ctx context.Context, invocationId uuid.UUID, status Status) error {
	_, err := s.db.Exec(ctx, `
		UPDATE invocations
		SET status = $1
		WHERE invocation_id = $2
	`, status, invocationId)
	if err != nil {
		return fmt.Errorf("update invocation status: %w", err)
	}
	return nil
}

func (s *Store) UpdateInvocationStatusIf(ctx context.Context, invocationId uuid.UUID, from, to Status) (bool, error) {
	result, err := s.db.Exec(ctx, `
		UPDATE invocations
		SET status = $1
		WHERE invocation_id = $2 AND status = $3
	`, to, invocationId, from)
	if err != nil {
		return false, fmt.Errorf("update invocation status: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (s *Store) UpdateInvocationStatusAndLogPathAndEndTime(ctx context.Context, invocationId uuid.UUID, status Status, logPath string, endTime time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE invocations
		SET status = $1, log_path = $2, end_time = $3
		WHERE invocation_id = $4
	`, status, logPath, endTime, invocationId)
	if err != nil {
		return fmt.Errorf("update invocation status and end time: %w", err)
	}
	return nil
}
