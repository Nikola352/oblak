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

func (s *Store) CreateFunction(ctx context.Context, userId uuid.UUID, status Status, archivePath string) (Function, error) {
	f := Function{
		FunctionId:  uuid.New(),
		UserId:      userId,
		Status:      status,
		ArchivePath: archivePath,
		DrivePath:   nil,
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO functions (function_id, user_id, status, archive_path, drive_path)
		VALUES ($1, $2, $3, $4, $5)
	`, f.FunctionId, f.UserId, f.Status, f.ArchivePath, f.DrivePath)
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

func (s *Store) UpdateFunctionStatusAndDrivePath(ctx context.Context, functionId uuid.UUID, status Status, drivePath string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE functions 
		SET status = $1,
		drive_path = $2
		WHERE function_id = $3
	`, status, drivePath, functionId)
	if err != nil {
		return fmt.Errorf("update status function: %w", err)
	}
	return nil
}

func (s *Store) GetFunctionsByUserWithLatestExecution(ctx context.Context, userUUID uuid.UUID) ([]FunctionWithLatestInvocation, error) {
	// DISTINCT ON guarantees exactly one row per function_id.
	// We sort by function_id first (required by DISTINCT ON) and then created_at DESC to get the newest execution.
	query := `
    SELECT 
        f.function_id, 
        f.user_id, 
        f.status, 
        latest_inv.invocation_id,
        latest_inv.status AS invocation_status,
        latest_inv.end_time AS invocation_time
    FROM functions f
    LEFT JOIN (
        SELECT DISTINCT ON (function_id) 
            function_id, 
            invocation_id, 
            status, 
            end_time
        FROM invocations
        ORDER BY function_id, end_time DESC
    ) latest_inv ON f.function_id = latest_inv.function_id
    WHERE f.user_id = $1
`
	rows, err := s.db.Query(ctx, query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("get functions with execution query: %w", err)
	}

	var functions []FunctionWithLatestInvocation
	for rows.Next() {
		var f FunctionWithLatestInvocation
		err := rows.Scan(
			&f.FunctionId,
			&f.UserId,
			&f.Status,
			&f.LatestInvocationId,     // Left Join means these can scan cleanly into pointers
			&f.LatestInvocationStatus, // if they return NULL from the database
			&f.LatestInvocationTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("get functions with execution scan: %w", err)
		}
		functions = append(functions, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get functions with execution rows loop: %w", err)
	}

	return functions, nil
}
func (s *Store) GetFunctionsByUserId(ctx context.Context, userUUID uuid.UUID) ([]Function, error) {
	rows, err := s.db.Query(ctx, `
		SELECT function_id, user_id, status, archive_path, drive_path
		FROM functions
		WHERE user_id = $1
	`, userUUID)
	if err != nil {
		return nil, fmt.Errorf("get functions by user id query: %w", err)
	}
	defer rows.Close()

	var functions []Function
	for rows.Next() {
		var f Function
		err := rows.Scan(
			&f.FunctionId,
			&f.UserId,
			&f.Status,
			&f.ArchivePath,
			&f.DrivePath,
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
