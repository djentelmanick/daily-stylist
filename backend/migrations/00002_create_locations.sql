-- +goose Up
CREATE TABLE locations (
    user_id   bigint           PRIMARY KEY,
    name      text             NOT NULL,
    region    text             NOT NULL,
    latitude  double precision NOT NULL,
    longitude double precision NOT NULL,
    timezone  text             NOT NULL
);

-- +goose Down
DROP TABLE locations;
