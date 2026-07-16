-- +goose Up
-- Multi-PLMN devices report comma-separated MCC+MNC values. Six five-digit
-- entries require 35 characters, which does not fit the original varchar(32).
ALTER TABLE public.device_info
    ALTER COLUMN plmn TYPE varchar(40);

-- +goose Down
-- Keep rollback non-destructive: rows written after Up may exceed 32 chars.
SELECT 1;
