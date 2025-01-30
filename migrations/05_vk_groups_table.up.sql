CREATE TABLE IF NOT EXISTS vk_groups (
    id      BIGINT  PRIMARY KEY,
    name    TEXT    NOT NULL,
    domain  TEXT    NOT NULL
)