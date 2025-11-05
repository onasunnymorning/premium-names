-- Teardown schema (reverse order to respect FKs)
DROP TABLE IF EXISTS label_tag;
DROP TABLE IF EXISTS tag;
DROP TABLE IF EXISTS batch_label;
DROP TABLE IF EXISTS label;
DROP TABLE IF EXISTS batch;

-- Extensions
DROP EXTENSION IF EXISTS citext;
