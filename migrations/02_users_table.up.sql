CREATE TABLE IF NOT EXISTS users
(
    id          BIGSERIAL   PRIMARY KEY,
    username    TEXT        UNIQUE,
    first_name  TEXT,
    last_name   TEXT,
    role_id     INTEGER     NOT NULL,
    FOREIGN KEY (role_id)   REFERENCES roles (id)   ON DELETE CASCADE
);

INSERT INTO users (id, username, role_id) VALUES
(5222179967, 'Her72hbf', 2);

