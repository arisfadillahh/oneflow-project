DELETE FROM organization_members om
WHERE om.organization_id = (
    SELECT id
    FROM organizations
    WHERE slug = 'default-company'
    LIMIT 1
  )
  AND om.agent_id IN (
    SELECT a.id
    FROM agents a
    WHERE a.role::text <> 'owner'
      AND EXISTS (
        SELECT 1
        FROM organization_members other_om
        JOIN organizations other_o ON other_o.id = other_om.organization_id
        WHERE other_om.agent_id = a.id
          AND other_om.status = 'active'
          AND other_o.status = 'active'
          AND other_o.slug <> 'default-company'
      )
  );
