ALTER TABLE knowledge_faqs
  ADD COLUMN IF NOT EXISTS embedding vector(1536);

ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS embedding vector(1536);
