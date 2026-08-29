-- Unprefixed public square IDs that were price_unset because defaults
-- only had nvidia/ or meta/ prefixed keys.
-- Token ratio: 1 = $2 / 1M input.
--
-- Production docker-compose uses PostgreSQL. JSON merge is idempotent.

-- ========== PostgreSQL ==========
UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"bevformer":0.05,"cosmos-transfer1-7b":0.1,"cosmos-transfer2.5-2b":0.05,"cosmos3-nano":0.03,"cosmos3-nano-reasoner":0.03,"ising-calibration-1-35b-a3b":0.03,"ising-calibration-1.5-31b":0.2,"llama-3.1-nemotron-safety-guard-8b-v3":0.03,"llama-guard-4-12b":0.1}'::jsonb)::text
WHERE "key" = 'ModelRatio';

UPDATE options
SET value = (COALESCE(NULLIF(btrim(value), ''), '{}')::jsonb || '{"cosmos3-nano":3.8333333333,"cosmos3-nano-reasoner":3.8333333333,"ising-calibration-1-35b-a3b":4,"llama-3.1-nemotron-safety-guard-8b-v3":3.8333333333}'::jsonb)::text
WHERE "key" = 'CompletionRatio';

-- ========== MySQL 5.7.8+ / MariaDB ==========
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"bevformer":0.05,"cosmos-transfer1-7b":0.1,"cosmos-transfer2.5-2b":0.05,"cosmos3-nano":0.03,"cosmos3-nano-reasoner":0.03,"ising-calibration-1-35b-a3b":0.03,"ising-calibration-1.5-31b":0.2,"llama-3.1-nemotron-safety-guard-8b-v3":0.03,"llama-guard-4-12b":0.1}' AS JSON))
-- WHERE `key` = 'ModelRatio';
-- UPDATE options
-- SET value = JSON_MERGE_PATCH(CAST(value AS JSON), CAST('{"cosmos3-nano":3.8333333333,"cosmos3-nano-reasoner":3.8333333333,"ising-calibration-1-35b-a3b":4,"llama-3.1-nemotron-safety-guard-8b-v3":3.8333333333}' AS JSON))
-- WHERE `key` = 'CompletionRatio';

-- ========== SQLite 3.38+ ==========
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"bevformer":0.05,"cosmos-transfer1-7b":0.1,"cosmos-transfer2.5-2b":0.05,"cosmos3-nano":0.03,"cosmos3-nano-reasoner":0.03,"ising-calibration-1-35b-a3b":0.03,"ising-calibration-1.5-31b":0.2,"llama-3.1-nemotron-safety-guard-8b-v3":0.03,"llama-guard-4-12b":0.1}')
-- WHERE key = 'ModelRatio';
-- UPDATE options
-- SET value = json_patch(ifnull(nullif(trim(value), ''), '{}'), '{"cosmos3-nano":3.8333333333,"cosmos3-nano-reasoner":3.8333333333,"ising-calibration-1-35b-a3b":4,"llama-3.1-nemotron-safety-guard-8b-v3":3.8333333333}')
-- WHERE key = 'CompletionRatio';
