WITH default_org AS (
  SELECT id
  FROM organizations
  WHERE slug = 'default-company'
  LIMIT 1
),
preferred_org AS (
  SELECT DISTINCT ON (om.agent_id)
    om.agent_id,
    om.organization_id
  FROM organization_members om
  JOIN organizations o ON o.id = om.organization_id
  WHERE om.status = 'active'
    AND o.status = 'active'
    AND o.slug <> 'default-company'
  ORDER BY om.agent_id, om.joined_at DESC
)
UPDATE agents a
SET current_organization_id = preferred_org.organization_id,
    updated_at = NOW()
FROM preferred_org, default_org
WHERE a.id = preferred_org.agent_id
  AND a.role::text <> 'owner'
  AND a.current_organization_id = default_org.id;

WITH default_org AS (
  SELECT id
  FROM organizations
  WHERE slug = 'default-company'
  LIMIT 1
),
preferred_org AS (
  SELECT DISTINCT om.agent_id
  FROM organization_members om
  JOIN organizations o ON o.id = om.organization_id
  WHERE om.status = 'active'
    AND o.status = 'active'
    AND o.slug <> 'default-company'
)
DELETE FROM organization_members om
USING default_org, preferred_org, agents a
WHERE om.organization_id = default_org.id
  AND om.agent_id = preferred_org.agent_id
  AND a.id = om.agent_id
  AND a.role::text <> 'owner';
