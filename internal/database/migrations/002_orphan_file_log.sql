-- Migration: 002_orphan_file_log.sql
-- Description: Create orphan_file_log table for tracking duplicate/orphan files detected and quarantined

CREATE TABLE IF NOT EXISTS orphan_file_log (
    id BIGSERIAL PRIMARY KEY,
    original_path VARCHAR(500) NOT NULL,
    quarantine_path VARCHAR(500),
    size BIGINT DEFAULT 0,
    action VARCHAR(50) NOT NULL, -- e.g., 'DRY_RUN', 'QUARANTINED', 'RESTORED', 'DELETED'
    reason VARCHAR(255) DEFAULT 'Orphan File - Not in DB',
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orphan_log_action ON orphan_file_log (action);
CREATE INDEX IF NOT EXISTS idx_orphan_log_detected_at ON orphan_file_log (detected_at);
