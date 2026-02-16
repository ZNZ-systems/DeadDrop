package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
)

type InboundIngestJobStore struct {
	db *sql.DB
}

func NewInboundIngestJobStore(db *sql.DB) *InboundIngestJobStore {
	return &InboundIngestJobStore{db: db}
}

func (s *InboundIngestJobStore) EnqueueInboundIngestJob(ctx context.Context, payload []byte, maxAttempts int) (*models.InboundIngestJob, error) {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	job := &models.InboundIngestJob{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO inbound_ingest_jobs (payload, max_attempts)
		 VALUES ($1, $2)
		 RETURNING id, status, payload, attempts, max_attempts, available_at, locked_at, last_error, accepted, dropped, created_at, updated_at, done_at`,
		payload, maxAttempts,
	).Scan(
		&job.ID, &job.Status, &job.Payload, &job.Attempts, &job.MaxAttempts,
		&job.AvailableAt, &job.LockedAt, &job.LastError, &job.Accepted, &job.Dropped,
		&job.CreatedAt, &job.UpdatedAt, &job.DoneAt,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *InboundIngestJobStore) ClaimNextInboundIngestJob(ctx context.Context) (*models.InboundIngestJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	job := &models.InboundIngestJob{}
	err = tx.QueryRowContext(ctx,
		`WITH next_job AS (
			SELECT id
			FROM inbound_ingest_jobs
			WHERE status = 'queued'
			  AND available_at <= NOW()
			ORDER BY available_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE inbound_ingest_jobs j
		SET status = 'processing',
			attempts = j.attempts + 1,
			locked_at = NOW(),
			updated_at = NOW()
		FROM next_job
		WHERE j.id = next_job.id
		RETURNING j.id, j.status, j.payload, j.attempts, j.max_attempts, j.available_at, j.locked_at, j.last_error, j.accepted, j.dropped, j.created_at, j.updated_at, j.done_at`,
	).Scan(
		&job.ID, &job.Status, &job.Payload, &job.Attempts, &job.MaxAttempts,
		&job.AvailableAt, &job.LockedAt, &job.LastError, &job.Accepted, &job.Dropped,
		&job.CreatedAt, &job.UpdatedAt, &job.DoneAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			if commitErr := tx.Commit(); commitErr != nil {
				return nil, commitErr
			}
			return nil, nil
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *InboundIngestJobStore) MarkInboundIngestJobDone(ctx context.Context, jobID int64, accepted, dropped int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE inbound_ingest_jobs
		 SET status = 'done',
		     accepted = $2,
		     dropped = $3,
		     last_error = '',
		     done_at = NOW(),
		     locked_at = NULL,
		     updated_at = NOW()
		 WHERE id = $1`,
		jobID, accepted, dropped,
	)
	return err
}

func (s *InboundIngestJobStore) MarkInboundIngestJobRetry(ctx context.Context, jobID int64, nextAvailableAt time.Time, lastError string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE inbound_ingest_jobs
		 SET status = 'queued',
		     available_at = $2,
		     last_error = $3,
		     locked_at = NULL,
		     updated_at = NOW()
		 WHERE id = $1`,
		jobID, nextAvailableAt, lastError,
	)
	return err
}

func (s *InboundIngestJobStore) MarkInboundIngestJobFailed(ctx context.Context, jobID int64, lastError string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE inbound_ingest_jobs
		 SET status = 'failed',
		     last_error = $2,
		     done_at = NOW(),
		     locked_at = NULL,
		     updated_at = NOW()
		 WHERE id = $1`,
		jobID, lastError,
	)
	return err
}
