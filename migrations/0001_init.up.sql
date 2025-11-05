-- Schema for labels + tags + batches
-- Extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- Batches (each upload)
CREATE TABLE IF NOT EXISTS batch (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    source_filename TEXT,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Labels (normalized once per unique label)
CREATE TABLE IF NOT EXISTS label (
    id BIGSERIAL PRIMARY KEY,
    label_ascii TEXT NOT NULL UNIQUE, -- punycoded, lowercased
    label_unicode TEXT NOT NULL,      -- display
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Membership of labels in batches (so the same label can appear in multiple uploads)
CREATE TABLE IF NOT EXISTS batch_label (
    batch_id BIGINT REFERENCES batch(id) ON DELETE CASCADE,
    label_id BIGINT REFERENCES label(id) ON DELETE CASCADE,
    pos INT,
    meta JSONB DEFAULT '{}'::jsonb,
    PRIMARY KEY (batch_id, label_id)
);

-- Tags and many-to-many binding (tags are global by default)
CREATE TABLE IF NOT EXISTS tag (
    id BIGSERIAL PRIMARY KEY,
    name CITEXT NOT NULL UNIQUE, -- case-insensitive
    group_name TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS label_tag (
    label_id BIGINT REFERENCES label(id) ON DELETE CASCADE,
    tag_id BIGINT REFERENCES tag(id) ON DELETE CASCADE,
    added_by TEXT,
    added_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (label_id, tag_id)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_label_ascii ON label (label_ascii);
CREATE INDEX IF NOT EXISTS idx_labeltag_tag ON label_tag (tag_id);
CREATE INDEX IF NOT EXISTS idx_batchlabel_batch ON batch_label (batch_id);
