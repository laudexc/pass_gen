CREATE TABLE IF NOT EXISTS password_hashes (
    id BIGSERIAL PRIMARY KEY,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS generation_audit (
    id BIGSERIAL PRIMARY KEY,
    password_length INT NOT NULL,
    password_count INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
