-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    target_count integer;
BEGIN
    SELECT count(*)
      INTO target_count
      FROM timescaledb_information.jobs
     WHERE proc_name = 'policy_retention'
       AND hypertable_schema = 'public'
       AND hypertable_name = 'alarms_history';

    IF target_count <> 1 THEN
        RAISE EXCEPTION 'expected exactly one alarms_history retention job, found %', target_count;
    END IF;
END
$$;

SELECT alter_job(
    j.job_id,
    schedule_interval => INTERVAL '1 day',
    fixed_schedule => TRUE,
    initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08',
    timezone => 'Asia/Shanghai'
)
  FROM timescaledb_information.jobs j
 WHERE j.proc_name = 'policy_retention'
   AND j.hypertable_schema = 'public'
   AND j.hypertable_name = 'alarms_history';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    target_count integer;
BEGIN
    SELECT count(*)
      INTO target_count
      FROM timescaledb_information.jobs
     WHERE proc_name = 'policy_retention'
       AND hypertable_schema = 'public'
       AND hypertable_name = 'alarms_history';

    IF target_count <> 1 THEN
        RAISE EXCEPTION 'expected exactly one alarms_history retention job, found %', target_count;
    END IF;
END
$$;

SELECT alter_job(
    j.job_id,
    schedule_interval => INTERVAL '1 day',
    fixed_schedule => FALSE,
    timezone => NULL
)
  FROM timescaledb_information.jobs j
 WHERE j.proc_name = 'policy_retention'
   AND j.hypertable_schema = 'public'
   AND j.hypertable_name = 'alarms_history';
-- TimescaleDB 保留的 initial_start 在 fixed_schedule=false 后不再参与调度。
-- 重建 policy 只为清空该元数据会改变 job_id，增加不必要的回滚风险。
-- +goose StatementEnd
