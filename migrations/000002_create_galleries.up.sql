BEGIN;

CREATE TABLE IF NOT EXISTS galleries (
    id          BIGSERIAL    NOT NULL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    published   BOOLEAN      NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL,
    user_id     INTEGER      NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS images (
    id         BIGSERIAL NOT NULL PRIMARY KEY,
    filepath   TEXT      NOT NULL,
    title      TEXT      NOT NULL,
    caption    TEXT      NOT NULL,
    size       BIGINT    NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL,
    gallery_id INTEGER   NOT NULL,

    CONSTRAINT image_to_galleries_fk FOREIGN KEY (gallery_id) REFERENCES galleries (id)
);

COMMIT;