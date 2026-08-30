-- Remaining unprefixed square IDs that were still price_unset after
-- bin/add_unprefixed_catalog_pricing.sql. Token ratio: 1 = $2 / 1M input.
-- Live Riva snapshot ID uses an underscore: riva-translate-4b-instruct-v1_1.
--
-- Production docker-compose uses PostgreSQL. JSON merge is idempotent.

-- ========== PostgreSQL ==========
UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"nemotron-3.5-content-safety":0.03,"riva-translate-4b-instruct-v1_1":0.02,"riva-translate-4b-instruct-v2":0.02,"sparsedrive":0.05,"streampetr":0.05,"synthetic-video-detector":0.05}'::jsonb)::text
WHERE "key" = 'ModelRatio';

UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"nemotron-3.5-content-safety":3.8333333333,"riva-translate-4b-instruct-v1_1":4,"riva-translate-4b-instruct-v2":4}'::jsonb)::text
WHERE "key" = 'CompletionRatio';

-- ========== MySQL 5.7.8+ / MariaDB ==========
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"nemotron-3.5-content-safety":0.03,"riva-translate-4b-instruct-v1_1":0.02,"riva-translate-4b-instruct-v2":0.02,"sparsedrive":0.05,"streampetr":0.05,"synthetic-video-detector":0.05}' AS JSON))
-- WHERE `key` = 'ModelRatio';
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"nemotron-3.5-content-safety":3.8333333333,"riva-translate-4b-instruct-v1_1":4,"riva-translate-4b-instruct-v2":4}' AS JSON))
-- WHERE `key` = 'CompletionRatio';

-- ========== SQLite 3.38+ ==========
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"nemotron-3.5-content-safety":0.03,"riva-translate-4b-instruct-v1_1":0.02,"riva-translate-4b-instruct-v2":0.02,"sparsedrive":0.05,"streampetr":0.05,"synthetic-video-detector":0.05}')
-- WHERE key = 'ModelRatio';
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"nemotron-3.5-content-safety":3.8333333333,"riva-translate-4b-instruct-v1_1":4,"riva-translate-4b-instruct-v2":4}')
-- WHERE key = 'CompletionRatio';
