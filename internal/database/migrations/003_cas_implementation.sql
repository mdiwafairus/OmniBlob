-- Migration: 003_cas_implementation.sql
-- Description: Add physical_objects table and app_name to binary_file for CAS implementation

CREATE TABLE IF NOT EXISTS public.physical_objects (
    content_hash    char(64) NOT NULL,
    "size"          int8 DEFAULT 0 NULL,
    storage_path    varchar(500) NOT NULL,
    first_stored_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT physical_objects_pkey PRIMARY KEY (content_hash)
);

ALTER TABLE public.binary_file ADD COLUMN IF NOT EXISTS app_name varchar(100) NULL;
CREATE INDEX IF NOT EXISTS idx_binary_file_app_name ON public.binary_file (app_name);
