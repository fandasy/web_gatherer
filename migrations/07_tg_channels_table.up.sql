CREATE TABLE IF NOT EXISTS tg_channels (
    id              BIGINT          PRIMARY KEY,
    name            VARCHAR(100)    NOT NULL,
    description     TEXT
);

CREATE INDEX IF NOT EXISTS tg_channel_idx ON tg_channels(id);