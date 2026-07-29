-- +goose Up
-- Repair every persisted enabled-indicator set, including operator-specific sets
-- changed before dependency-closure validation was introduced.
WITH RECURSIVE dependency_closure(operator_code, indicator_id) AS (
    SELECT operator_code, indicator_id
    FROM public.enabled_pm_indicators_enb
    UNION
    SELECT closure.operator_code, dependency.id
    FROM dependency_closure AS closure
    JOIN public.perf_indicators_enb AS parent ON parent.id = closure.indicator_id
    CROSS JOIN LATERAL regexp_matches(
        COALESCE(parent.arithmetic, ''),
        '([CK][A-Za-z0-9_.]+)',
        'g'
    ) AS parsed(dependency_id)
    JOIN public.perf_indicators_enb AS dependency
      ON lower(dependency.id) = lower(parsed.dependency_id[1])
)
INSERT INTO public.enabled_pm_indicators_enb (operator_code, indicator_id)
SELECT operator_code, indicator_id
FROM dependency_closure
ON CONFLICT (operator_code, indicator_id) DO NOTHING;

WITH RECURSIVE dependency_closure(operator_code, indicator_id) AS (
    SELECT operator_code, indicator_id
    FROM public.enabled_pm_indicators_gnb
    UNION
    SELECT closure.operator_code, dependency.id
    FROM dependency_closure AS closure
    JOIN public.perf_indicators_gnb AS parent ON parent.id = closure.indicator_id
    CROSS JOIN LATERAL regexp_matches(
        COALESCE(parent.arithmetic, ''),
        '([CK][A-Za-z0-9_.]+)',
        'g'
    ) AS parsed(dependency_id)
    JOIN public.perf_indicators_gnb AS dependency
      ON lower(dependency.id) = lower(parsed.dependency_id[1])
)
INSERT INTO public.enabled_pm_indicators_gnb (operator_code, indicator_id)
SELECT operator_code, indicator_id
FROM dependency_closure
ON CONFLICT (operator_code, indicator_id) DO NOTHING;

WITH RECURSIVE dependency_closure(operator_code, indicator_id) AS (
    SELECT operator_code, indicator_id
    FROM public.enabled_pm_indicators_gsm
    UNION
    SELECT closure.operator_code, dependency.id
    FROM dependency_closure AS closure
    JOIN public.perf_indicators_gsm AS parent ON parent.id = closure.indicator_id
    CROSS JOIN LATERAL regexp_matches(
        COALESCE(parent.arithmetic, ''),
        '([CK][A-Za-z0-9_.]+)',
        'g'
    ) AS parsed(dependency_id)
    JOIN public.perf_indicators_gsm AS dependency
      ON lower(dependency.id) = lower(parsed.dependency_id[1])
)
INSERT INTO public.enabled_pm_indicators_gsm (operator_code, indicator_id)
SELECT operator_code, indicator_id
FROM dependency_closure
ON CONFLICT (operator_code, indicator_id) DO NOTHING;

-- K900010076 is the dashboard output. C000060216 and C000060273 are formula
-- inputs and are retained by the closure above, but must not replace the KPI.
UPDATE public.pm_tasks
SET metric_paths = array_replace(metric_paths, 'C000060216', 'K900010076'),
    updated_at = now()
WHERE id IN (
    '0184dddd-0001-4000-8000-000000000001',
    '0184dddd-0002-4000-8000-000000000001',
    '0184dddd-0003-4000-8000-000000000001',
    '0184dddd-0004-4000-8000-000000000001'
)
  AND metric_paths @> ARRAY['C000060216']::text[]
  AND NOT metric_paths @> ARRAY['K900010076']::text[];

UPDATE public.pm_tasks
SET metric_paths = array_remove(metric_paths, 'C000060216'),
    updated_at = now()
WHERE id IN (
    '0184dddd-0001-4000-8000-000000000001',
    '0184dddd-0002-4000-8000-000000000001',
    '0184dddd-0003-4000-8000-000000000001',
    '0184dddd-0004-4000-8000-000000000001'
)
  AND metric_paths @> ARRAY['C000060216', 'K900010076']::text[];

-- +goose Down
UPDATE public.pm_tasks
SET metric_paths = array_replace(metric_paths, 'K900010076', 'C000060216'),
    updated_at = now()
WHERE id IN (
    '0184dddd-0001-4000-8000-000000000001',
    '0184dddd-0002-4000-8000-000000000001',
    '0184dddd-0003-4000-8000-000000000001',
    '0184dddd-0004-4000-8000-000000000001'
)
  AND metric_paths @> ARRAY['K900010076']::text[]
  AND NOT metric_paths @> ARRAY['C000060216']::text[];

