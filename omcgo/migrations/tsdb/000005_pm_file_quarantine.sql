-- +goose Up
CREATE TABLE IF NOT EXISTS public.pm_file_quarantines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL,
    device_sn text NOT NULL,
    declared_technology varchar(16) NOT NULL,
    detected_technology varchar(16) NOT NULL,
    reason varchar(64) NOT NULL,
    minio_path text NOT NULL,
    evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_pm_file_quarantines_source_reason
        UNIQUE (source_file_id, reason)
);

CREATE INDEX IF NOT EXISTS idx_pm_file_quarantines_created_at
    ON public.pm_file_quarantines (created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS public.idx_pm_file_quarantines_created_at;
DROP TABLE IF EXISTS public.pm_file_quarantines;
