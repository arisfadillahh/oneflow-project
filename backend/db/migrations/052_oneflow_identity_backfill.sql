UPDATE agents
SET role = 'operator'::agent_role, updated_at = NOW()
WHERE role::text = 'hr_agent';

UPDATE organization_members
SET role = 'operator', updated_at = NOW()
WHERE role = 'hr_agent';

UPDATE organization_members
SET role = 'supervisor', updated_at = NOW()
WHERE role = 'hr_manager';

UPDATE organization_invites
SET role = 'operator', updated_at = NOW()
WHERE role = 'hr_agent';

UPDATE organization_invites
SET role = 'supervisor', updated_at = NOW()
WHERE role = 'hr_manager';

UPDATE messages
SET sender_type = 'customer'::message_sender_type
WHERE sender_type::text = 'candidate'
  AND direction::text = 'inbound';
