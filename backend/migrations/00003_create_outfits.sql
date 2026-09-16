-- +goose Up
CREATE TABLE outfits (
    id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL,
    worn_on date   NOT NULL,
    UNIQUE (user_id, worn_on)
);

CREATE TABLE outfit_items (
    outfit_id bigint NOT NULL REFERENCES outfits (id) ON DELETE CASCADE,
    item_id   bigint NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    PRIMARY KEY (outfit_id, item_id)
);

CREATE INDEX outfit_items_item_id_idx ON outfit_items (item_id);

-- +goose Down
DROP TABLE outfit_items;
DROP TABLE outfits;
