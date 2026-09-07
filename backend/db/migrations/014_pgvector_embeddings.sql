CREATE EXTENSION IF NOT EXISTS vector;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'knowledge_chunks'
      AND column_name = 'embedding'
      AND data_type = 'jsonb'
  ) THEN
    ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_vector vector(1536);
    UPDATE knowledge_chunks
    SET embedding_vector = embedding::text::vector(1536)
    WHERE embedding IS NOT NULL
      AND jsonb_typeof(embedding) = 'array'
      AND jsonb_array_length(embedding) = 1536;
    ALTER TABLE knowledge_chunks DROP COLUMN embedding;
    ALTER TABLE knowledge_chunks RENAME COLUMN embedding_vector TO embedding;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'knowledge_faqs'
      AND column_name = 'embedding'
      AND data_type = 'jsonb'
  ) THEN
    ALTER TABLE knowledge_faqs ADD COLUMN IF NOT EXISTS embedding_vector vector(1536);
    UPDATE knowledge_faqs
    SET embedding_vector = embedding::text::vector(1536)
    WHERE embedding IS NOT NULL
      AND jsonb_typeof(embedding) = 'array'
      AND jsonb_array_length(embedding) = 1536;
    ALTER TABLE knowledge_faqs DROP COLUMN embedding;
    ALTER TABLE knowledge_faqs RENAME COLUMN embedding_vector TO embedding;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'job_positions'
      AND column_name = 'embedding'
      AND data_type = 'jsonb'
  ) THEN
    ALTER TABLE job_positions ADD COLUMN IF NOT EXISTS embedding_vector vector(1536);
    UPDATE job_positions
    SET embedding_vector = embedding::text::vector(1536)
    WHERE embedding IS NOT NULL
      AND jsonb_typeof(embedding) = 'array'
      AND jsonb_array_length(embedding) = 1536;
    ALTER TABLE job_positions DROP COLUMN embedding;
    ALTER TABLE job_positions RENAME COLUMN embedding_vector TO embedding;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'knowledge_records'
      AND column_name = 'embedding'
      AND data_type = 'jsonb'
  ) THEN
    ALTER TABLE knowledge_records ADD COLUMN IF NOT EXISTS embedding_vector vector(1536);
    UPDATE knowledge_records
    SET embedding_vector = embedding::text::vector(1536)
    WHERE embedding IS NOT NULL
      AND jsonb_typeof(embedding) = 'array'
      AND jsonb_array_length(embedding) = 1536;
    ALTER TABLE knowledge_records DROP COLUMN embedding;
    ALTER TABLE knowledge_records RENAME COLUMN embedding_vector TO embedding;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_hnsw
ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_knowledge_faqs_embedding_hnsw
ON knowledge_faqs USING hnsw (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_job_positions_embedding_hnsw
ON job_positions USING hnsw (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_knowledge_records_embedding_hnsw
ON knowledge_records USING hnsw (embedding vector_cosine_ops);
