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

func (s *Store) GetInvocationsForUserAndFunction(ctx context.Context, userId, functionId uuid.UUID) ([]Invocation, error) {
	query := `
SELECT i.invocation_id, i.function_id, i.status, i.end_time, i.invocation_time
from invocations i join functions f on i.function_id = f.function_id
where f.user_id = $1 and i.function_id = $2 ORDER BY i.end_time
`
	rows, err := s.db.Query(ctx, query, userId, functionId)
	if err != nil {
		return nil, fmt.Errorf("get functions with execution query: %w", err)
	}
	defer rows.Close()
	var functions []Invocation
	for rows.Next() {
		var f Invocation
		err := rows.Scan(
			&f.InvocationId,
			&f.FunctionId,
			&f.Status,
			&f.EndTime,
			&f.InvocationTime,
		)
		if err != nil {
			return nil, fmt.Errorf("get functions by user id scan: %w", err)
		}
		functions = append(functions, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get functions by user id rows loop: %w", err)
	}

	return functions, nil
}
