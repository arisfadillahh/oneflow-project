ALTER TABLE IF EXISTS contact_memory_candidates
  RENAME TO contact_memory_drafts;

ALTER INDEX IF EXISTS idx_contact_memory_candidates_org_contact_created
  RENAME TO idx_contact_memory_drafts_org_contact_created;

ALTER TABLE IF EXISTS contact_memories
  RENAME COLUMN source_candidate_id TO source_draft_id;
