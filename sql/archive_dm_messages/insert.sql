INSERT INTO archive_dm_messages (guild_id, ticket_id, message_id)
VALUES ($1, $2, $3)
ON CONFLICT (guild_id, ticket_id) DO UPDATE SET
    message_id = excluded.message_id;
