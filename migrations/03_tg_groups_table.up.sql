CREATE TABLE IF NOT EXISTS tg_groups (
    id              BIGINT          PRIMARY KEY,
    name            VARCHAR(100)    NOT NULL,
    description     TEXT
);

CREATE INDEX IF NOT EXISTS tg_group_idx ON tg_groups(id);