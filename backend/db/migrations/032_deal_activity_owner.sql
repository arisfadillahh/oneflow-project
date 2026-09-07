ALTER TABLE deal_activities
  ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES agents(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_deal_activities_org_assignee_due
  ON deal_activities (organization_id, assigned_to, due_at)
  WHERE activity_type = 'task' AND completed_at IS NULL;

