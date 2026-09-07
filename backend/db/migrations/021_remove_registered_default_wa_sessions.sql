DELETE FROM whatsapp_sessions ws
USING organizations o
WHERE ws.organization_id = o.id
  AND o.slug <> 'default-company'
  AND ws.label = 'Default WhatsApp'
  AND ws.session_key = o.id::text || ':default'
  AND ws.status = 'disconnected'
  AND NOT EXISTS (
    SELECT 1
    FROM conversations c
    WHERE c.whatsapp_session_id = ws.id
  );
