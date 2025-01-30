CREATE TABLE IF NOT EXISTS tg_group_messages (
    msg_id      BIGINT      NOT NULL,
    group_id    BIGINT      NOT NULL,
    username    TEXT        NOT NULL,
    text        TEXT        NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (msg_id, group_id),
    FOREIGN KEY (group_id)  REFERENCES tg_groups (id)  ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE OR REPLACE FUNCTION notify_insert_tg_group_msg()
    RETURNS TRIGGER AS $$
DECLARE
    msg_json TEXT;
BEGIN
    -- Создаем JSON-объект с нужными полями
    msg_json := json_build_object(
            'group_name', (SELECT g.name FROM tg_groups g WHERE g.id = NEW.group_id),
            'username', NEW.username,
            'text', NEW.text,
            'metadata', NEW.metadata,
            'created_at', to_char(NEW.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
                )::text;

    -- Отправляем уведомление с подготовленным JSON
    PERFORM pg_notify('insert_tg_group_message', msg_json);

    RETURN NEW;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER insert_tg_group_msg_trigger
    AFTER INSERT ON tg_group_messages
    FOR EACH ROW
EXECUTE PROCEDURE notify_insert_tg_group_msg();
