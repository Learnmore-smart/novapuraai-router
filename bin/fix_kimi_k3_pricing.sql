-- Correct Kimi K3 to Moonshot's official token prices:
--   cache-miss input: $3 / 1M  => ModelRatio 3 / 2 = 1.5
--   output:          $15 / 1M  => CompletionRatio 15 / 3 = 5
--   cache-hit input: $0.30 / 1M => CacheRatio 0.30 / 3 = 0.1
--
-- JSON updates are idempotent and preserve every other model entry.
-- Production docker-compose uses PostgreSQL; run only the matching dialect.

-- ========== PostgreSQL ========== 
UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"kimi-k3":1.5}'::jsonb)::text
WHERE "key" = 'ModelRatio';

UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"kimi-k3":5}'::jsonb)::text
WHERE "key" = 'CompletionRatio';

UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"kimi-k3":0.1}'::jsonb)::text
WHERE "key" = 'CacheRatio';

-- ========== MySQL 5.7.8+ / MariaDB ========== 
-- UPDATE options
-- SET value = JSON_SET(COALESCE(NULLIF(TRIM(value), ''), '{}'), '$."kimi-k3"', 1.5)
-- WHERE `key` = 'ModelRatio';
-- UPDATE options
-- SET value = JSON_SET(COALESCE(NULLIF(TRIM(value), ''), '{}'), '$."kimi-k3"', 5)
-- WHERE `key` = 'CompletionRatio';
-- UPDATE options
-- SET value = JSON_SET(COALESCE(NULLIF(TRIM(value), ''), '{}'), '$."kimi-k3"', 0.1)
-- WHERE `key` = 'CacheRatio';

-- ========== SQLite 3.38+ ========== 
-- UPDATE options
-- SET value = json_set(ifnull(nullif(trim(value), ''), '{}'), '$."kimi-k3"', 1.5)
-- WHERE key = 'ModelRatio';
-- UPDATE options
-- SET value = json_set(ifnull(nullif(trim(value), ''), '{}'), '$."kimi-k3"', 5)
-- WHERE key = 'CompletionRatio';
-- UPDATE options
-- SET value = json_set(ifnull(nullif(trim(value), ''), '{}'), '$."kimi-k3"', 0.1)
-- WHERE key = 'CacheRatio';
