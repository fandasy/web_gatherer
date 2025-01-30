CREATE TABLE IF NOT EXISTS web_messages (
    id          SERIAL  PRIMARY KEY,
    group_name  TEXT    NOT NULL,
    text        TEXT    NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMP NOT NULL,
    type        TEXT
)