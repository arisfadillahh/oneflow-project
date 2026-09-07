CREATE TABLE IF NOT EXISTS knowledge_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  record_type TEXT NOT NULL DEFAULT 'generic',
  title TEXT NOT NULL,
  content TEXT,
  fields_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  source_type TEXT NOT NULL DEFAULT 'record',
  priority INTEGER NOT NULL DEFAULT 70,
  locked BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'published',
  topic TEXT,
  intent TEXT,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  embedding vector(1536),
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_records_source_metadata
ON knowledge_records (status, locked DESC, priority DESC, intent, topic, record_type);

INSERT INTO knowledge_records (
  id,
  record_type,
  title,
  content,
  fields_json,
  source_type,
  priority,
  locked,
  status,
  topic,
  intent,
  metadata_json,
  embedding,
  created_by,
  updated_by,
  created_at,
  updated_at
)
SELECT
  id,
  'job_position',
  title,
  COALESCE(short_description, ''),
  jsonb_strip_nulls(jsonb_build_object(
    'status', status,
    'location', location,
    'work_type', work_type,
    'minimum_education', minimum_education,
    'minimum_experience', minimum_experience,
    'short_description', short_description,
    'apply_link', apply_link,
    'is_active', is_active
  )),
  'record',
  COALESCE(priority, 70),
  COALESCE(locked, FALSE),
  CASE WHEN is_active THEN 'published' ELSE 'draft' END,
  topic,
  intent,
  COALESCE(metadata_json, '{}'::jsonb) || jsonb_build_object('legacy_table', 'job_positions', 'legacy_source_type', COALESCE(source_type, '')),
  embedding,
  created_by,
  updated_by,
  created_at,
  updated_at
FROM job_positions
ON CONFLICT (id) DO UPDATE SET
  record_type = EXCLUDED.record_type,
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  fields_json = EXCLUDED.fields_json,
  source_type = EXCLUDED.source_type,
  priority = EXCLUDED.priority,
  locked = EXCLUDED.locked,
  status = EXCLUDED.status,
  topic = EXCLUDED.topic,
  intent = EXCLUDED.intent,
  metadata_json = EXCLUDED.metadata_json,
  embedding = EXCLUDED.embedding,
  updated_by = EXCLUDED.updated_by,
  updated_at = EXCLUDED.updated_at;
