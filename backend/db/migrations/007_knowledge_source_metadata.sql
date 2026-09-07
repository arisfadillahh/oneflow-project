ALTER TABLE knowledge_faqs
  ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'faq',
  ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 80,
  ADD COLUMN IF NOT EXISTS locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS topic TEXT,
  ADD COLUMN IF NOT EXISTS intent TEXT,
  ADD COLUMN IF NOT EXISTS metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE knowledge_documents
  ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'document',
  ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 60,
  ADD COLUMN IF NOT EXISTS locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS topic TEXT,
  ADD COLUMN IF NOT EXISTS intent TEXT,
  ADD COLUMN IF NOT EXISTS metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'job_position',
  ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 70,
  ADD COLUMN IF NOT EXISTS locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS topic TEXT,
  ADD COLUMN IF NOT EXISTS intent TEXT,
  ADD COLUMN IF NOT EXISTS metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_knowledge_faqs_source_metadata
  ON knowledge_faqs (status, locked DESC, priority DESC, intent, topic);

CREATE INDEX IF NOT EXISTS idx_knowledge_documents_source_metadata
  ON knowledge_documents (status, locked DESC, priority DESC, intent, topic);

CREATE INDEX IF NOT EXISTS idx_job_positions_source_metadata
  ON job_positions (is_active, locked DESC, priority DESC, intent, topic);

UPDATE knowledge_faqs
SET source_type = 'faq', priority = GREATEST(priority, 90)
WHERE status = 'published';

UPDATE knowledge_documents
SET source_type = 'admin_rules', priority = 5, topic = COALESCE(topic, 'system_rules')
WHERE knowledge_type = 'system_rules';

UPDATE knowledge_documents
SET source_type = 'document', priority = GREATEST(priority, 60)
WHERE knowledge_type <> 'system_rules';

UPDATE job_positions
SET source_type = 'job_position', priority = GREATEST(priority, 70)
WHERE is_active = TRUE;

UPDATE knowledge_faqs
SET topic = 'customer_process', intent = 'service_flow', priority = GREATEST(priority, 95)
WHERE question ILIKE '%tahapan layanan%';

UPDATE knowledge_documents
SET topic = 'customer_process', intent = 'service_flow', priority = GREATEST(priority, 95)
WHERE title ILIKE '%tahapan layanan%';

UPDATE knowledge_documents
SET topic = 'registration', intent = 'registration_steps'
WHERE title ILIKE '%proses pendaftaran%';
