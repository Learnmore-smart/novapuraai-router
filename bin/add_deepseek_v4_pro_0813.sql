-- Add official DeepSeek V4 Pro snapshot ID (unprefixed).
-- Client ID matches deepseek-v4-flash-0731: map to the NVIDIA NIM slug.
-- Price: $0.66 / $1.98 per 1M (DeepSeek official off-peak)
--   model_ratio = 0.66 / 2 = 0.33
--   completion_ratio = 1.98 / 0.66 = 3
-- Same token price as already-hosted deepseek-ai/deepseek-v4-pro-0813.
--
-- Production docker-compose uses PostgreSQL. Run the matching dialect, then
-- wait for channel/option sync (~60s) so /api/pricing and routing refresh.
-- JSON merge keeps existing priced models; CSV append is idempotent.

-- ========== PostgreSQL ==========
UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"deepseek-v4-pro-0813":0.33}'::jsonb)::text
WHERE "key" = 'ModelRatio';

UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"deepseek-v4-pro-0813":3}'::jsonb)::text
WHERE "key" = 'CompletionRatio';

-- Exact CSV token match so we do not confuse deepseek-ai/deepseek-v4-pro-0813
-- with the unprefixed official ID.
UPDATE channels
SET models = CASE
  WHEN models IS NULL OR btrim(models) = '' THEN 'deepseek-v4-pro-0813'
  WHEN right(btrim(models), 1) = ',' THEN btrim(models) || 'deepseek-v4-pro-0813'
  ELSE btrim(models) || ',deepseek-v4-pro-0813'
END
WHERE (',' || models || ',') LIKE '%,deepseek-v4-flash-0731,%'
  AND (',' || models || ',') NOT LIKE '%,deepseek-v4-pro-0813,%';

UPDATE channels
SET model_mapping = (
  COALESCE(NULLIF(btrim(model_mapping), ''), '{}')::jsonb
  || '{"deepseek-v4-pro-0813":"deepseek-ai/deepseek-v4-pro-0813"}'::jsonb
)::text
WHERE (',' || models || ',') LIKE '%,deepseek-v4-pro-0813,%'
  AND COALESCE(NULLIF(btrim(model_mapping), ''), '{}')::jsonb ->> 'deepseek-v4-pro-0813'
      IS DISTINCT FROM 'deepseek-ai/deepseek-v4-pro-0813';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, tag)
SELECT a."group", 'deepseek-v4-pro-0813', a.channel_id, a.enabled, a.priority, a.weight, a.tag
FROM abilities a
WHERE a.model = 'deepseek-v4-flash-0731'
ON CONFLICT ("group", model, channel_id) DO NOTHING;

-- ========== MySQL 5.7.8+ / MariaDB ==========
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"deepseek-v4-pro-0813":0.33}' AS JSON))
-- WHERE `key` = 'ModelRatio';
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"deepseek-v4-pro-0813":3}' AS JSON))
-- WHERE `key` = 'CompletionRatio';

-- ========== SQLite 3.38+ ==========
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"deepseek-v4-pro-0813":0.33}')
-- WHERE key = 'ModelRatio';
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"deepseek-v4-pro-0813":3}')
-- WHERE key = 'CompletionRatio';
