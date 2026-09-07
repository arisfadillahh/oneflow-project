ALTER TYPE agent_role ADD VALUE IF NOT EXISTS 'operator';
ALTER TYPE message_sender_type ADD VALUE IF NOT EXISTS 'customer';

ALTER TABLE organization_members DROP CONSTRAINT IF EXISTS organization_members_role_check;
ALTER TABLE organization_members
  ADD CONSTRAINT organization_members_role_check
  CHECK (role IN ('org_owner', 'org_admin', 'supervisor', 'operator', 'hr_manager', 'hr_agent'));

ALTER TABLE organization_invites DROP CONSTRAINT IF EXISTS organization_invites_role_check;
ALTER TABLE organization_invites
  ADD CONSTRAINT organization_invites_role_check
  CHECK (role IN ('org_owner', 'org_admin', 'supervisor', 'operator', 'hr_manager', 'hr_agent'));
