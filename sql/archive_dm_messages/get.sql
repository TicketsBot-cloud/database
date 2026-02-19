SELECT message_id
FROM archive_dm_messages
WHERE guild_id = $1 AND ticket_id = $2;
