-- +goose Up
CREATE TABLE items (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      bigint      NOT NULL,
    name         text        NOT NULL,
    description  text        NOT NULL,
    category     text        NOT NULL,
    main_color   text        NOT NULL,
    extra_colors text[]      NOT NULL,
    seasons      text[]      NOT NULL,
    warmth_level smallint    NOT NULL,
    waterproof   boolean     NOT NULL,
    status       text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX items_user_id_idx ON items (user_id);

-- +goose Down
DROP TABLE items;
