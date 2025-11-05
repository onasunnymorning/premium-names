package db

import (
	"context"
	"errors"
)

// Job represents an import job record.
type Job struct {
	ID        int64
	BatchID   int64
	ObjectURI string
	Status    string
	Error     *string
	StatsJSON []byte
}

type JobRepository interface {
	Enqueue(ctx context.Context, batchID int64, objectURI string) (Job, error)
	// ClaimNext attempts to atomically claim the next queued job using SKIP LOCKED; returns ErrNotFound if none.
	ClaimNext(ctx context.Context) (Job, error)
	UpdateStatus(ctx context.Context, id int64, status string, errMsg *string, statsJSON []byte) error
}

func NewJobRepo(p *Pool) JobRepository { return &jobRepo{p: p} }

type jobRepo struct{ p *Pool }

func (r *jobRepo) Enqueue(ctx context.Context, batchID int64, objectURI string) (Job, error) {
	const q = `insert into job (batch_id, object_uri, status) values ($1, $2, 'queued')
               returning id, batch_id, object_uri, status, error, coalesce(stats,'{}'::jsonb)`
	var j Job
	err := r.p.QueryRow(ctx, q, batchID, objectURI).Scan(&j.ID, &j.BatchID, &j.ObjectURI, &j.Status, &j.Error, &j.StatsJSON)
	if err != nil {
		return Job{}, mapPgErr(err)
	}
	return j, nil
}

func (r *jobRepo) ClaimNext(ctx context.Context) (Job, error) {
	// Advisory: limit to a single row using SKIP LOCKED so multiple workers can process concurrently.
	const q = `with cte as (
                  select id from job where status='queued' order by id asc for update skip locked limit 1
               )
               update job j set status='running'
               from cte where j.id = cte.id
               returning j.id, j.batch_id, j.object_uri, j.status, j.error, coalesce(j.stats,'{}'::jsonb)`
	var j Job
	err := r.p.QueryRow(ctx, q).Scan(&j.ID, &j.BatchID, &j.ObjectURI, &j.Status, &j.Error, &j.StatsJSON)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Job{}, ErrNotFound
		}
		return Job{}, mapPgErr(err)
	}
	return j, nil
}

func (r *jobRepo) UpdateStatus(ctx context.Context, id int64, status string, errMsg *string, statsJSON []byte) error {
	const q = `update job set status=$1, error=$2, stats=coalesce($3::jsonb, stats) where id=$4`
	_, err := r.p.Exec(ctx, q, status, errMsg, string(statsJSON), id)
	return err
}
