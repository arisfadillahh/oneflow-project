UPDATE knowledge_documents
SET
  raw_text = $RULES$
# AI Routing and Grounding Rules

Use this document only as routing and behavior guidance. Do not use it as a factual answer source.

## Source Priority

1. Prefer locked or admin-approved knowledge when it directly answers the user's latest message.
2. Prefer user-facing sources such as FAQ, documents, and records over internal rules when they answer the question.
3. Use internal rules only to decide behavior, escalation, formatting, and source selection.
4. If no source contains enough information, escalate instead of inventing details.

## Grounding

- Answer only from selected knowledge, structured records, and conversation memory.
- Do not add timelines, procedures, channels, prices, availability, requirements, or promises unless they are explicitly present in the selected source.
- If the user asks about a specific personal status and the information is not present in conversation memory or selected knowledge, escalate.
- If sources conflict, prefer locked or higher-priority sources and mention only the supported answer.

## Conversation Handling

- Treat the latest user message as the main query.
- Use previous messages only to resolve references such as "that", "the previous one", or "same as above".
- Do not let an older topic override the latest user message.

## Response Style

- Use natural, concise language.
- Keep the answer easy to read in chat.
- Preserve important facts from the selected source.
- Do not expose internal source labels, metadata, or routing decisions to the user.
$RULES$,
  source_type = 'admin_rules',
  priority = 5,
  topic = 'system_rules',
  intent = NULL,
  updated_at = NOW()
WHERE knowledge_type = 'system_rules'
  AND source_type IN ('admin_rules', 'system_rules');

DELETE FROM knowledge_chunks
WHERE knowledge_document_id IN (
  SELECT id
  FROM knowledge_documents
  WHERE knowledge_type = 'system_rules'
    AND source_type IN ('admin_rules', 'system_rules')
);

INSERT INTO knowledge_chunks (knowledge_document_id, chunk_index, chunk_text, embedding, metadata_json)
SELECT
  id,
  0,
  raw_text,
  NULL,
  jsonb_build_object('source', 'admin_routing_rules', 'chunking', 'manual_single_chunk')
FROM knowledge_documents
WHERE knowledge_type = 'system_rules'
  AND source_type IN ('admin_rules', 'system_rules');
