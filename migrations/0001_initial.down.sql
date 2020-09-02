BEGIN;

DROP TRIGGER set_updated_timestamp_builds ON builds;
DROP FUNCTION trigger_set_updated_timestamp();
DROP TABLE builds;
DROP TABLE api_keys;

COMMIT;