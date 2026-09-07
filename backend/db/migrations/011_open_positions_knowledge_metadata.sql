UPDATE knowledge_documents
SET
  topic = 'available_services',
  intent = 'list_available_options',
  priority = GREATEST(priority, 98),
  source_type = 'document',
  updated_at = NOW()
WHERE status = 'published'
  AND title ILIKE '%Layanan Tersedia%';

UPDATE knowledge_records
SET
  topic = 'available_services',
  intent = 'list_available_options',
  priority = GREATEST(priority, 96),
  source_type = 'record',
  status = 'published',
  updated_at = NOW()
WHERE metadata_json->>'legacy_table' = 'job_positions';

UPDATE knowledge_faqs
SET
  topic = 'available_services',
  intent = 'availability_summary',
  priority = GREATEST(priority, 97)
WHERE status = 'published'
  AND question ILIKE '%benar sedang ada layanan%';

UPDATE knowledge_faqs
SET
  topic = 'available_services',
  intent = 'specific_position_availability',
  priority = LEAST(priority, 90)
WHERE status = 'published'
  AND (
    question ILIKE '%layanan konsultasi%'
    OR question ILIKE '%layanan support%'
  );
