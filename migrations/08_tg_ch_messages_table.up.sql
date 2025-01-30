CREATE TABLE IF NOT EXISTS tg_channel_messages (
    msg_id      BIGINT      NOT NULL,
    channel_id  BIGINT      NOT NULL,
    text        TEXT        NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (msg_id, channel_id),
    FOREIGN KEY (channel_id)  REFERENCES tg_channels (id)  ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE OR REPLACE FUNCTION notify_insert_tg_channel_msg()
    RETURNS TRIGGER AS $$
DECLARE
    msg_json TEXT;
BEGIN
    -- Создаем JSON-объект с нужными полями
    msg_json := json_build_object(
            'channel_name', (SELECT g.name FROM tg_channels g WHERE g.id = NEW.channel_id),
            'text', NEW.text,
            'metadata', NEW.metadata,
            'created_at', to_char(NEW.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
                )::text;

    -- Отправляем уведомление с подготовленным JSON
    PERFORM pg_notify('insert_tg_channel_message', msg_json);

    RETURN NEW;
END;
$$
    LANGUAGE plpgsql;

CREATE TRIGGER insert_tg_channel_msg_trigger
    AFTER INSERT ON tg_channel_messages
    FOR EACH ROW
EXECUTE PROCEDURE notify_insert_tg_channel_msg();