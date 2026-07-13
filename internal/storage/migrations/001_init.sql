CREATE TABLE IF NOT EXISTS links (
    code VARCHAR(8) PRIMARY KEY,
    original_url TEXT NOT NULL CHECK (length(original_url) <= 4096),
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS links_expires_at_idx ON links (expires_at) WHERE expires_at IS NOT NULL;
