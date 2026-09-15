-- Migration: 002_migration_history.sql
-- Description: Create job_history table to record sync and migration jobs

CREATE TABLE IF NOT EXISTS job_history (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    volume_bytes BIGINT DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    client_id VARCHAR(100) DEFAULT '',
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_history_type ON job_history(type);
CREATE INDEX IF NOT EXISTS idx_job_history_client ON job_history(client_id);
CREATE INDEX IF NOT EXISTS idx_job_history_time ON job_history(end_time DESC);
