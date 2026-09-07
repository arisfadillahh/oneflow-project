UPDATE knowledge_faqs
SET
  topic = 'registration',
  intent = 'registration_steps',
  priority = GREATEST(priority, 98),
  source_type = 'faq'
WHERE status = 'published'
  AND (
    question ILIKE '%cara daftar%'
    OR (
      answer ILIKE '%CARA MENDAFTAR%'
      AND answer ILIKE '%website resmi%'
      AND answer ILIKE '%submit%'
    )
  );

UPDATE knowledge_documents
SET
  topic = 'registration',
  intent = 'registration_steps',
  priority = GREATEST(priority, 96)
WHERE status = 'published'
  AND COALESCE(knowledge_type, '') <> 'system_rules'
  AND (
    title ILIKE '%pendaftaran%'
    OR (
      raw_text ILIKE '%CARA MENDAFTAR%'
      AND raw_text ILIKE '%website resmi%'
      AND raw_text ILIKE '%submit%'
    )
  );
