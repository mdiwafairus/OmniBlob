-- Migration: 001_init.sql
-- Description: Create log_file_rsync and ensure binary_file has required columns

CREATE TABLE IF NOT EXISTS log_file_rsync (
    id BIGSERIAL PRIMARY KEY,
    server VARCHAR(100) NOT NULL,
    last_bin_id BIGINT DEFAULT 0,
    bin_executed JSONB,
    error_logs TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS binary_file (
    bin_id BIGSERIAL PRIMARY KEY,
    referensi_id VARCHAR(100),
    module VARCHAR(100) DEFAULT '',
    directory VARCHAR(255) DEFAULT '',
    file_name VARCHAR(255),
    path VARCHAR(500),
    size BIGINT DEFAULT 0,
    mime_type VARCHAR(100) DEFAULT '',
    checksum VARCHAR(64) DEFAULT '',
    flag VARCHAR(10) DEFAULT '1',
    create_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Safely add columns if existing binary_file table lacks them
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS module VARCHAR(100) DEFAULT '';
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS directory VARCHAR(255) DEFAULT '';
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS size BIGINT DEFAULT 0;
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS mime_type VARCHAR(100) DEFAULT '';
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS checksum VARCHAR(64) DEFAULT '';
ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS flag VARCHAR(10) DEFAULT '1';

CREATE INDEX IF NOT EXISTS idx_binary_file_referensi_id ON binary_file (referensi_id);
CREATE INDEX IF NOT EXISTS idx_binary_file_module ON binary_file (module);
CREATE INDEX IF NOT EXISTS idx_log_file_rsync_server ON log_file_rsync (server);
