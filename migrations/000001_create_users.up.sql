BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id                      BIGSERIAL    NOT NULL,
    name                    VARCHAR(255) NOT NULL,
    email                   VARCHAR(255) NOT NULL UNIQUE,
    password_hash           TEXT         NOT NULL,
    activated               BOOL         NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP    NOT NULL DEFAULT NOW(),
    version                 INTEGER      NOT NULL DEFAULT 1,

    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS tokens (
    hash        TEXT                  NOT NULL,
    scope       TEXT                  NOT NULL,
    expiry      TIMESTAMP             NOT NULL,
    created_at  TIMESTAMP             NOT NULL DEFAULT NOW(),
    user_id     BIGINT                NOT NULL,

    PRIMARY KEY (hash),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auth_keys (
    id                  BIGSERIAL    NOT NULL,
    auth_key_hash       TEXT         NOT NULL UNIQUE,
    created_at          TIMESTAMP    NOT NULL DEFAULT NOW(),
    user_id             BIGINT       NOT NULL,

    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS permissions (
    id   BIGSERIAL     NOT NULL,
    code TEXT          NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS auth_keys_permissions (
    auth_key_id     BIGINT      NOT NULL,
    permission_id   BIGINT      NOT NULL,

    PRIMARY KEY (auth_key_id, permission_id),
    FOREIGN KEY (auth_key_id) REFERENCES auth_keys (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

INSERT INTO permissions (code) VALUES
    ('*:*'),
    ('admin'),

    ('keys:list'),
    ('keys:create'),
    ('keys:update'),
    ('keys:delete'),
    ('users:stats'),

    ('galleries:list'),
    ('galleries:create'),
    ('galleries:update'),
    ('galleries:delete'),
    ('galleries:download'),

    ('images:list'),
    ('images:create'),
    ('images:update'),
    ('images:delete'),
    ('images:download')
;

COMMIT;