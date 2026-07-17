-- +goose Up
CREATE TABLE public.device_location_observations (
    -- devices is partitioned by carrier and its primary key is (id, carrier),
    -- so a single-column FK to devices(id) is not valid in PostgreSQL.
    -- Device ownership is validated by the repository before writes.
    device_id uuid PRIMARY KEY,
    latitude double precision NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude double precision NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    gps_height double precision,
    observed_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    source_path text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_location_observations_observed_at
    ON public.device_location_observations (device_id, observed_at DESC);

COMMENT ON TABLE public.device_location_observations IS '设备最新有效 GPS 观测值；不等同于网管已接受坐标';

-- +goose Down
DROP INDEX IF EXISTS public.idx_device_location_observations_observed_at;
DROP TABLE IF EXISTS public.device_location_observations;
