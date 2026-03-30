-- Job scheduler schema
-- Mirrors the Java jfc_job table structure

CREATE TABLE IF NOT EXISTS job_configs (
    job_name          VARCHAR(255) PRIMARY KEY,
    concurrency_rules JSONB        NOT NULL DEFAULT '{}',
    destination_type  VARCHAR(50)  NOT NULL DEFAULT 'IN_MEMORY',
    destination_config JSONB       NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS jobs (
    id                  BIGSERIAL PRIMARY KEY,
    job_name            VARCHAR(255)  NOT NULL,
    tenant              INTEGER       NOT NULL,
    priority            INTEGER       NOT NULL DEFAULT 5,
    status              VARCHAR(50)   NOT NULL DEFAULT 'WAITING',
    payload             JSONB         NOT NULL DEFAULT '{}',
    concurrency_control JSONB         NOT NULL DEFAULT '{}',
    execution_count     INTEGER       NOT NULL DEFAULT 0,
    last_failure_reason TEXT,
    submit_time         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    last_updated        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Indexes matching the Java @Index annotations
CREATE INDEX IF NOT EXISTS idx_jobs_submit_time    ON jobs (submit_time);
CREATE INDEX IF NOT EXISTS idx_jobs_name_status    ON jobs (job_name, status);
CREATE INDEX IF NOT EXISTS idx_jobs_status_name_tenant_updated
    ON jobs (status, job_name, tenant, last_updated);

-- Seed: sample job configs
INSERT INTO job_configs (job_name, concurrency_rules, destination_type)
VALUES
    ('DataIngestion',  '{"$jobName": 10, "$tenant_$env": 3}', 'IN_MEMORY'),
    ('ReportGeneration', '{"$jobName": 5, "$tenant": 2}',     'IN_MEMORY')
ON CONFLICT (job_name) DO NOTHING;
