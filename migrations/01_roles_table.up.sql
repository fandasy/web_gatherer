CREATE TABLE IF NOT EXISTS roles
(
    id      SERIAL         PRIMARY KEY,
    name    VARCHAR(30)    UNIQUE
);


INSERT INTO roles (name) VALUES
    ('sub user'),
    ('admin');

INSERT INTO roles(id, name) VALUES
    (0, 'system');