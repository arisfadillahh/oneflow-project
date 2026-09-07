CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS vector;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'conversation_mode') THEN
    CREATE TYPE conversation_mode AS ENUM ('ai', 'human');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'conversation_status') THEN
    CREATE TYPE conversation_status AS ENUM ('open', 'pending_human', 'resolved');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_sender_type') THEN
    CREATE TYPE message_sender_type AS ENUM ('candidate', 'ai', 'agent', 'system');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_direction') THEN
    CREATE TYPE message_direction AS ENUM ('inbound', 'outbound');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_content_type') THEN
    CREATE TYPE message_content_type AS ENUM ('text', 'image', 'document', 'audio', 'system');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'conversation_event_type') THEN
    CREATE TYPE conversation_event_type AS ENUM (
      'conversation_opened',
      'ai_replied',
      'escalated',
      'taken_over_by_human',
      'returned_to_ai',
      'conversation_resolved',
      'conversation_reopened',
      'assignment_changed',
      'force_taken_over',
      'manual_message_sent'
    );
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_actor_type') THEN
    CREATE TYPE event_actor_type AS ENUM ('system', 'ai', 'agent');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'agent_role') THEN
    CREATE TYPE agent_role AS ENUM ('owner', 'super_admin', 'admin', 'hr_agent');
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  phone TEXT,
  role agent_role NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone TEXT NOT NULL UNIQUE,
  name TEXT,
  email TEXT,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contact_id UUID NOT NULL UNIQUE REFERENCES contacts(id) ON DELETE CASCADE,
  channel TEXT NOT NULL DEFAULT 'whatsapp',
  mode conversation_mode NOT NULL DEFAULT 'ai',
  status conversation_status NOT NULL DEFAULT 'open',
  assigned_to UUID REFERENCES agents(id) ON DELETE SET NULL,
  last_message_id UUID,
  last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  escalation_reason TEXT,
  escalated_at TIMESTAMPTZ,
  human_taken_over_at TIMESTAMPTZ,
  returned_to_ai_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  external_message_id TEXT UNIQUE,
  sender_type message_sender_type NOT NULL,
  direction message_direction NOT NULL,
  content_type message_content_type NOT NULL,
  text TEXT,
  raw_payload JSONB,
  reply_to_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  sender_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
  sent_at TIMESTAMPTZ,
  delivered_at TIMESTAMPTZ,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  event_type conversation_event_type NOT NULL,
  actor_type event_actor_type,
  actor_id UUID,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_settings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  system_prompt TEXT NOT NULL,
  escalation_prompt TEXT NOT NULL,
  fallback_waiting_message TEXT NOT NULL,
  allow_clarification BOOLEAN NOT NULL DEFAULT TRUE,
  max_clarification_count INTEGER NOT NULL DEFAULT 1,
  answer_only_from_knowledge BOOLEAN NOT NULL DEFAULT TRUE,
  dont_broaden_topic BOOLEAN NOT NULL DEFAULT TRUE,
  forbid_promises BOOLEAN NOT NULL DEFAULT TRUE,
  forbid_sensitive_answers BOOLEAN NOT NULL DEFAULT TRUE,
  require_action_confirmation BOOLEAN NOT NULL DEFAULT TRUE,
  escalate_low_confidence BOOLEAN NOT NULL DEFAULT TRUE,
  guide_next_step BOOLEAN NOT NULL DEFAULT TRUE,
  concise_response BOOLEAN NOT NULL DEFAULT TRUE,
  allow_auto_update_contact_name BOOLEAN NOT NULL DEFAULT TRUE,
  only_fill_name_if_empty BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  decision TEXT NOT NULL,
  question_summary TEXT,
  answer_text TEXT,
  confidence_score NUMERIC(5,4),
  escalation_reason TEXT,
  retrieval_metadata JSONB,
  model_name TEXT,
  latency_ms INTEGER,
  ai_settings_updated_at_snapshot TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS knowledge_faqs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  question TEXT NOT NULL,
  answer TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'published',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS knowledge_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  knowledge_type TEXT NOT NULL,
  file_url TEXT,
  raw_text TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  knowledge_document_id UUID NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL,
  chunk_text TEXT NOT NULL,
  embedding vector(1536),
  metadata_json JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS job_positions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  location TEXT,
  work_type TEXT,
  minimum_education TEXT,
  minimum_experience TEXT,
  short_description TEXT,
  apply_link TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_wallet (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  monthly_credit_limit INTEGER NOT NULL DEFAULT 10000,
  monthly_credits_used INTEGER NOT NULL DEFAULT 0,
  monthly_credits_remaining INTEGER NOT NULL DEFAULT 10000,
  additional_credits_remaining INTEGER NOT NULL DEFAULT 0,
  last_reset_at TIMESTAMPTZ,
  next_reset_at TIMESTAMPTZ,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_usage_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  ai_run_id UUID REFERENCES ai_runs(id) ON DELETE SET NULL,
  usage_type TEXT NOT NULL,
  credit_source TEXT NOT NULL,
  model_name TEXT,
  input_tokens INTEGER,
  output_tokens INTEGER,
  embedding_tokens INTEGER,
  cost_usd NUMERIC(12,6),
  cost_idr NUMERIC(12,2),
  credits_used INTEGER NOT NULL DEFAULT 0,
  credit_unit_idr_snapshot NUMERIC(12,2) NOT NULL DEFAULT 500,
  usd_to_idr_rate_snapshot NUMERIC(12,2) NOT NULL DEFAULT 16000,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_packages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  credit_amount INTEGER NOT NULL,
  price NUMERIC(12,2) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_purchases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  credit_package_id UUID REFERENCES credit_packages(id) ON DELETE SET NULL,
  credit_amount INTEGER NOT NULL,
  price NUMERIC(12,2) NOT NULL,
  payment_method TEXT NOT NULL,
  payment_status TEXT NOT NULL,
  requested_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  confirmed_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  confirmed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS credit_adjustments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  adjustment_type TEXT NOT NULL,
  amount INTEGER NOT NULL,
  notes TEXT,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_pricing_settings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  credit_unit_idr NUMERIC(12,2) NOT NULL,
  usd_to_idr_rate NUMERIC(12,2) NOT NULL,
  chat_model_name TEXT NOT NULL,
  chat_input_price_per_1m NUMERIC(12,6) NOT NULL,
  chat_output_price_per_1m NUMERIC(12,6) NOT NULL,
  embedding_model_name TEXT NOT NULL,
  embedding_price_per_1m NUMERIC(12,6) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS system_status (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  service_name TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  details JSONB,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_at ON messages (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_events_conversation_created_at ON conversation_events (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_usage_logs_created_at ON credit_usage_logs (created_at DESC);
