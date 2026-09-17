-- +goose Up
ALTER TABLE items ADD COLUMN photo_key text NOT NULL DEFAULT '';

-- Ссылка на загрузку выдана, но фотография ещё ни к какой вещи не привязана.
-- Строка живёт от выдачи ссылки до сохранения вещи, потом удаляется.
CREATE TABLE photo_uploads (
    key        text        PRIMARY KEY,
    user_id    bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX photo_uploads_user_id_idx ON photo_uploads (user_id);

-- +goose Down
DROP TABLE photo_uploads;
ALTER TABLE items DROP COLUMN photo_key;
