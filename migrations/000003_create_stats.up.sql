BEGIN;

CREATE TABLE IF NOT EXISTS stats (
    n_galleries INTEGER NOT NULL,
    n_images    INTEGER NOT NULL,
    n_bytes     INTEGER NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    version     INTEGER NOT NULL DEFAULT 1,
    user_id     INTEGER NOT NULL UNIQUE,

    FOREIGN KEY (user_id) REFERENCES users (id)
);

COMMIT;