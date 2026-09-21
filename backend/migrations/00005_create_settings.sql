-- +goose Up
CREATE TABLE settings (
    user_id         bigint  PRIMARY KEY,
    morning_enabled boolean NOT NULL,
    send_at         time    NOT NULL
);

CREATE TABLE morning_deliveries (
    user_id bigint NOT NULL,
    day     date   NOT NULL,
    PRIMARY KEY (user_id, day)
);

-- +goose Down
DROP TABLE morning_deliveries;
DROP TABLE settings;
