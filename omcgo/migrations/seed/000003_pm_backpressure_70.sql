-- +goose Up

-- Keep the confirmed PM disk protection default at 70% without overwriting an
-- operator-customized threshold. Existing baseline 85/75 values are upgraded;
-- any other values are treated as intentional local configuration.
UPDATE sys_configs
   SET value='70', updated_at=now()
 WHERE category='acs.backpressure' AND key='disk_high_pct' AND value='85';

UPDATE sys_configs
   SET value='60', updated_at=now()
 WHERE category='acs.backpressure' AND key='disk_low_pct' AND value='75';

-- +goose Down
-- Preserve the safer 70/60 defaults; rollback must not weaken disk protection.
SELECT 1;
