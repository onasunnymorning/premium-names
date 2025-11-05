-- Jobs table for upload/import pipeline
create table if not exists job (
    id bigserial primary key,
    batch_id bigint not null references batch(id) on delete cascade,
    object_uri text not null,
    status text not null default 'queued', -- queued|running|done|failed
    error text,
    stats jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_job_status on job(status);

create or replace function job_touch_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

drop trigger if exists job_set_updated on job;
create trigger job_set_updated before update on job for each row execute procedure job_touch_updated_at();
