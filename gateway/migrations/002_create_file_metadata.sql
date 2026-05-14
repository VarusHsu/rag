CREATE TABLE IF NOT EXISTS file_metadata (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user"(id),
    bucket        VARCHAR(128) NOT NULL,
    object_key    TEXT NOT NULL UNIQUE,
    file_name     TEXT NOT NULL,
    content_type  VARCHAR(255) NOT NULL,
    file_size     BIGINT NOT NULL CHECK (file_size > 0),
    status        VARCHAR(30) NOT NULL DEFAULT 'pending_upload',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_file_metadata_user_id ON file_metadata(user_id);
CREATE INDEX IF NOT EXISTS idx_file_metadata_status ON file_metadata(status);

DROP TRIGGER IF EXISTS trg_file_metadata_set_updated_at ON file_metadata;

CREATE TRIGGER trg_file_metadata_set_updated_at
BEFORE UPDATE ON file_metadata
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

