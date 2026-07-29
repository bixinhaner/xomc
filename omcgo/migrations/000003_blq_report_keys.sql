-- +goose Up
ALTER TABLE public.rela_platform_indicator_formula_enb ADD COLUMN report_key text;
ALTER TABLE public.rela_platform_indicator_formula_gsm ADD COLUMN report_key text;
ALTER TABLE public.rela_platform_indicator_formula_gnb ADD COLUMN report_key text;

UPDATE public.rela_platform_indicator_formula_enb AS route
SET report_key = indicator.report_key
FROM public.perf_indicators_enb AS indicator
WHERE indicator.id = route.indicator_id;
UPDATE public.rela_platform_indicator_formula_gsm AS route
SET report_key = indicator.report_key
FROM public.perf_indicators_gsm AS indicator
WHERE indicator.id = route.indicator_id;
UPDATE public.rela_platform_indicator_formula_gnb AS route
SET report_key = indicator.report_key
FROM public.perf_indicators_gnb AS indicator
WHERE indicator.id = route.indicator_id;

UPDATE public.rela_platform_indicator_formula_enb
SET report_key = CASE indicator_id
        WHEN 'C000010070' THEN 'ERAB.EstabInitAttNbr.Sum'
        WHEN 'C000010080' THEN 'ERAB.EstabInitSuccNbr.Sum'
    END,
    updated_at = now()
WHERE platform_name = 'BLQ'
  AND indicator_id IN ('C000010070', 'C000010080');

-- +goose Down
ALTER TABLE public.rela_platform_indicator_formula_gnb DROP COLUMN IF EXISTS report_key;
ALTER TABLE public.rela_platform_indicator_formula_gsm DROP COLUMN IF EXISTS report_key;
ALTER TABLE public.rela_platform_indicator_formula_enb DROP COLUMN IF EXISTS report_key;
