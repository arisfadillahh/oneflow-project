CREATE INDEX IF NOT EXISTS idx_deal_activities_org_open_due
  ON deal_activities (organization_id, due_at)
  WHERE activity_type = 'task' AND completed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_deal_activities_org_type_created
  ON deal_activities (organization_id, activity_type, created_at DESC);
