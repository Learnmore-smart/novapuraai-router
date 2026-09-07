-- Keep the admin model catalog aligned with enabled channel abilities.
-- Production is PostgreSQL. Do not import adaptor ModelList / official catalogs.
--
-- Also drops unused Kimi K2.6 (NVIDIA slug moonshotai/kimi-k2.6) from
-- channels.models / abilities / model_mapping. CSV token removal is applied
-- by the operator script because comma-list surgery is safer in application code.

-- ========== PostgreSQL ==========
-- Keep the upstream auto-sync task from re-adding K2.6 after this cleanup.
UPDATE channels
SET settings = jsonb_set(
  COALESCE(NULLIF(btrim(settings), ''), '{}')::jsonb,
  '{upstream_model_update_ignored_models}',
  COALESCE(
    COALESCE(NULLIF(btrim(settings), ''), '{}')::jsonb -> 'upstream_model_update_ignored_models',
    '[]'::jsonb
  ) || '["moonshotai/kimi-k2.6","kimi-k2.6"]'::jsonb,
  true
)::text
WHERE (',' || models || ',') LIKE '%,moonshotai/kimi-k2.6,%'
   OR (',' || models || ',') LIKE '%,kimi-k2.6,%';

DELETE FROM abilities
WHERE model IN ('moonshotai/kimi-k2.6', 'kimi-k2.6');

UPDATE channels
SET model_mapping = (
  COALESCE(NULLIF(btrim(model_mapping), ''), '{}')::jsonb
  - 'moonshotai/kimi-k2.6'
  - 'kimi-k2.6'
)::text
WHERE COALESCE(NULLIF(btrim(model_mapping), ''), '{}')::jsonb ?| ARRAY['moonshotai/kimi-k2.6', 'kimi-k2.6'];

INSERT INTO models (model_name, status, sync_official, name_rule, created_time, updated_time)
SELECT DISTINCT a.model, 1, 0, 0,
       EXTRACT(EPOCH FROM NOW())::bigint,
       EXTRACT(EPOCH FROM NOW())::bigint
FROM abilities a
WHERE a.enabled = TRUE
  AND a.model NOT IN ('moonshotai/kimi-k2.6', 'kimi-k2.6')
  AND NOT EXISTS (
    SELECT 1 FROM models m
    WHERE m.model_name = a.model AND m.deleted_at IS NULL
  );
