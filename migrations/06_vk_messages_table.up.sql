CREATE TABLE IF NOT EXISTS vk_messages (
    msg_id      BIGINT      NOT NULL,
    group_id    BIGINT      NOT NULL,
    text        TEXT        NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (msg_id, group_id),
    FOREIGN KEY (group_id)  REFERENCES vk_groups (id)  ON DELETE CASCADE ON UPDATE CASCADE
    );

CREATE OR REPLACE FUNCTION notify_insert_vk_msg()
    RETURNS TRIGGER AS $$
DECLARE
    msg_json TEXT;
BEGIN
    -- Создаем JSON-объект с нужными полями
    msg_json := json_build_object(
            'group_name', (SELECT g.name FROM vk_groups g WHERE g.id = NEW.group_id),
            'text', NEW.text,
            'metadata', NEW.metadata,
            'created_at', to_char(NEW.created_at AT TIME ZONE 'Europe/Moscow', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
                )::text;

    -- Отправляем уведомление с подготовленным JSON
    PERFORM pg_notify('insert_vk_message', msg_json);

    RETURN NEW;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER insert_vk_msg_trigger
    AFTER INSERT ON vk_messages
    FOR EACH ROW
EXECUTE PROCEDURE notify_insert_vk_msg();