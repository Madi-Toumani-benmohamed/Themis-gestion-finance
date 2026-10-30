package intelligence

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const maxJobAttempts = 5

type Job struct {
	OwnerID  int64
	Trigger  string
	Attempts int
}

type JobStatus struct {
	Status        string     `json:"status"`
	Trigger       string     `json:"trigger"`
	RequestedAt   time.Time  `json:"requested_at"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	Attempts      int        `json:"attempts"`
}

// Enqueue coalesces repeated mutations. A request arriving during a run asks
// for exactly one additional pass after the current snapshot finishes.
func Enqueue(ctx context.Context, db *sql.DB, ownerID int64, trigger string) error {
	if ownerID <= 0 || len(trigger) < 1 || len(trigger) > 80 {
		return errors.New("invalid intelligence job")
	}
	_, err := db.ExecContext(ctx, `INSERT INTO intelligence_jobs(owner_id,status,trigger,requested_at,available_at,attempts,rerun_requested,last_error,updated_at)
 VALUES($1,'queued',$2,now(),now(),0,false,'',now())
 ON CONFLICT(owner_id) DO UPDATE SET
 status=CASE WHEN intelligence_jobs.status='running' THEN 'running' ELSE 'queued' END,
 trigger=EXCLUDED.trigger,requested_at=now(),available_at=CASE WHEN intelligence_jobs.status='running' THEN intelligence_jobs.available_at ELSE now() END,
 attempts=CASE WHEN intelligence_jobs.status='running' THEN intelligence_jobs.attempts ELSE 0 END,
 rerun_requested=CASE WHEN intelligence_jobs.status='running' THEN true ELSE false END,
 last_error='',updated_at=now()`, ownerID, trigger)
	return err
}

func Claim(ctx context.Context, db *sql.DB) (*Job, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE intelligence_jobs SET
 status=CASE WHEN rerun_requested THEN 'queued' ELSE 'failed' END,
 attempts=CASE WHEN rerun_requested THEN 0 ELSE attempts END,
 available_at=CASE WHEN rerun_requested THEN now() ELSE available_at END,
 rerun_requested=false,lease_until=NULL,last_error='Le recalcul intelligence a échoué; une nouvelle tentative sera effectuée si disponible.',updated_at=now()
 WHERE status='running' AND lease_until<now() AND attempts >= $1`, maxJobAttempts); err != nil {
		return nil, err
	}
	var job Job
	err = tx.QueryRowContext(ctx, `WITH candidate AS (
 SELECT owner_id FROM intelligence_jobs
 WHERE (status='queued' AND available_at<=now()) OR (status='running' AND lease_until<now())
 ORDER BY requested_at,owner_id FOR UPDATE SKIP LOCKED LIMIT 1
)
 UPDATE intelligence_jobs j SET status='running',lease_until=now()+interval '5 minutes',attempts=j.attempts+1,rerun_requested=false,updated_at=now()
 FROM candidate c WHERE j.owner_id=c.owner_id
 RETURNING j.owner_id,j.trigger,j.attempts`).Scan(&job.OwnerID, &job.Trigger, &job.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &job, nil
}

func RenewJob(ctx context.Context, db *sql.DB, ownerID int64) error {
	_, err := db.ExecContext(ctx, `UPDATE intelligence_jobs SET lease_until=now()+interval '5 minutes',updated_at=now() WHERE owner_id=$1 AND status='running'`, ownerID)
	return err
}

func CompleteJob(ctx context.Context, db *sql.DB, ownerID int64) error {
	_, err := db.ExecContext(ctx, `UPDATE intelligence_jobs SET
 status=CASE WHEN rerun_requested THEN 'queued' ELSE 'done' END,
 available_at=CASE WHEN rerun_requested THEN now() ELSE available_at END,
 attempts=CASE WHEN rerun_requested THEN 0 ELSE attempts END,
 lease_until=NULL,rerun_requested=false,last_success_at=now(),last_error='',updated_at=now()
 WHERE owner_id=$1 AND status='running'`, ownerID)
	return err
}

// FailJob stores only a generic error so SQL details and financial data never
// enter a user-visible job status or long-lived log.
func FailJob(ctx context.Context, db *sql.DB, ownerID int64, attempts int) error {
	backoff := time.Duration(attempts*15) * time.Second
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	_, err := db.ExecContext(ctx, `UPDATE intelligence_jobs SET
 status=CASE WHEN rerun_requested THEN 'queued' WHEN attempts >= $2 THEN 'failed' ELSE 'queued' END,
 available_at=CASE WHEN rerun_requested OR attempts >= $2 THEN now() ELSE now()+$3::interval END,
 attempts=CASE WHEN rerun_requested THEN 0 ELSE attempts END,
 lease_until=NULL,rerun_requested=false,last_error='Le recalcul intelligence a échoué; une nouvelle tentative sera effectuée si disponible.',updated_at=now()
 WHERE owner_id=$1 AND status='running'`, ownerID, maxJobAttempts, backoff.String())
	return err
}

func GetJobStatus(ctx context.Context, db *sql.DB, ownerID int64) (JobStatus, error) {
	if ownerID <= 0 {
		return JobStatus{}, errors.New("invalid intelligence job owner")
	}
	var out JobStatus
	err := db.QueryRowContext(ctx, `SELECT status,trigger,requested_at,last_success_at,last_error,attempts FROM intelligence_jobs WHERE owner_id=$1`, ownerID).Scan(&out.Status, &out.Trigger, &out.RequestedAt, &out.LastSuccessAt, &out.LastError, &out.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return JobStatus{Status: "idle"}, nil
	}
	return out, err
}
