package seed

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type agentSeed struct {
	Name     string
	Username string
	Password string
	Role     string
}

var agents = []agentSeed{
	{Name: "Platform Owner", Username: "owner", Password: "owner123", Role: "owner"},
	{Name: "Super Admin", Username: "superadmin", Password: "admin123", Role: "super_admin"},
	{Name: "Operations Admin", Username: "admin", Password: "admin123", Role: "admin"},
	{Name: "Support Agent", Username: "supportagent", Password: "agent123", Role: "operator"},
}

func Bootstrap(ctx context.Context, pool *pgxpool.Pool) error {
	if demoAccountsEnabled() {
		if err := seedDemoAgents(ctx, pool); err != nil {
			return err
		}
	} else if err := seedInitialOwner(ctx, pool); err != nil {
		return err
	}
	if err := seedDefaultOrganizationMembership(ctx, pool); err != nil {
		return err
	}

	_, err := pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO ai_settings (
		  system_prompt,
		  escalation_prompt,
		  fallback_waiting_message,
		  allow_clarification,
		  max_clarification_count,
		  answer_only_from_knowledge,
		  dont_broaden_topic,
		  forbid_promises,
		  forbid_sensitive_answers,
		  allow_auto_update_contact_name,
		  only_fill_name_if_empty,
		  is_active,
		  organization_id
		)
		SELECT
		  'You are the company AI chatbot assistant. Answer only from approved business knowledge.',
		  'Escalate if confidence is low, sensitive, or outside approved knowledge.',
		  'Tim kami sedang meninjau pertanyaan Anda. Mohon tunggu sebentar ya.',
		  TRUE, 1, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE,
		  (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM ai_settings WHERE is_active = TRUE AND organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO credit_wallet (
		  monthly_credit_limit,
		  monthly_credits_used,
		  monthly_credits_remaining,
		  additional_credits_remaining,
		  last_reset_at,
		  next_reset_at,
		  is_active,
		  organization_id
		)
		SELECT 10000, 1240, 8760, 2500, NOW(), NOW() + INTERVAL '30 days', TRUE, (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM credit_wallet WHERE organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO credit_pricing_settings (
		  credit_unit_idr,
		  usd_to_idr_rate,
		  chat_model_name,
		  chat_input_price_per_1m,
		  chat_output_price_per_1m,
		  embedding_model_name,
		  embedding_price_per_1m,
		  is_active,
		  organization_id
		)
		SELECT 500, 16000, 'openai/gpt-4o-mini', 0.15, 0.60, 'openai/text-embedding-3-small', 0.02, TRUE, (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM credit_pricing_settings WHERE is_active = TRUE AND organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO credit_packages (name, credit_amount, price, is_active, organization_id)
		SELECT 'Starter Top Up', 5000, 2500000, TRUE, (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM credit_packages WHERE organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH admin_user AS (
		  SELECT id FROM agents WHERE username = 'admin'
		), default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO knowledge_faqs (question, answer, status, created_by, updated_by, published_at, organization_id)
		SELECT
		  'Kapan tim operasional akan menghubungi saya?',
		  'Tim kami akan menghubungi pelanggan sesuai kebutuhan layanan atau pembelian.',
		  'published',
		  (SELECT id FROM admin_user),
		  (SELECT id FROM admin_user),
		  NOW(),
		  (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM knowledge_faqs WHERE organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH admin_user AS (
		  SELECT id FROM agents WHERE username = 'admin'
		), default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO knowledge_documents (title, knowledge_type, raw_text, status, created_by, updated_by, published_at, organization_id)
		SELECT
		  'Business Policy Summary',
		  'text',
		  'Pricing negotiation and contract-specific promises must be escalated to a human agent.',
		  'published',
		  (SELECT id FROM admin_user),
		  (SELECT id FROM admin_user),
		  NOW(),
		  (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM knowledge_documents WHERE organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH admin_user AS (
		  SELECT id FROM agents WHERE username = 'admin'
		), default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO job_positions (
		  title,
		  status,
		  location,
		  work_type,
		  minimum_education,
		  minimum_experience,
		  short_description,
		  apply_link,
		  is_active,
		  created_by,
		  updated_by,
		  organization_id
		)
		SELECT
		  'Backend Engineer',
		  'published',
		  'Jakarta',
		  'hybrid',
		  'S1 Teknik Informatika atau setara',
		  '2 tahun',
		  'Menyediakan layanan chatbot AI untuk support, sales, dan operasional.',
		  'https://example.com/apply/backend',
		  TRUE,
		  (SELECT id FROM admin_user),
		  (SELECT id FROM admin_user),
		  (SELECT id FROM default_org)
		WHERE NOT EXISTS (SELECT 1 FROM job_positions WHERE organization_id = (SELECT id FROM default_org))
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH support_agent AS (
		  SELECT id FROM agents WHERE username = 'supportagent'
		), default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		), contact_a AS (
		  INSERT INTO contacts (phone, name, email, notes, organization_id)
		  SELECT '+628111111111', 'Budi Santoso', 'budi@example.com', 'Customer support contact', (SELECT id FROM default_org)
		  WHERE NOT EXISTS (SELECT 1 FROM contacts WHERE phone = '+628111111111' AND organization_id = (SELECT id FROM default_org))
		  RETURNING id
		), contact_b AS (
		  INSERT INTO contacts (phone, name, email, notes, organization_id)
		  SELECT '+628122222222', 'Siti Rahma', 'siti@example.com', 'Sales lead', (SELECT id FROM default_org)
		  WHERE NOT EXISTS (SELECT 1 FROM contacts WHERE phone = '+628122222222' AND organization_id = (SELECT id FROM default_org))
		  RETURNING id
		), resolved_contact AS (
		  INSERT INTO contacts (phone, name, email, notes, organization_id)
		  SELECT '+628133333333', 'Andi Wijaya', 'andi@example.com', 'Resolved customer conversation', (SELECT id FROM default_org)
		  WHERE NOT EXISTS (SELECT 1 FROM contacts WHERE phone = '+628133333333' AND organization_id = (SELECT id FROM default_org))
		  RETURNING id
		)
		INSERT INTO conversations (contact_id, mode, status, assigned_to, last_message_at, escalation_reason, escalated_at, human_taken_over_at, organization_id)
		SELECT c.id, 'ai'::conversation_mode, 'open'::conversation_status, NULL::uuid, NOW() - INTERVAL '5 minutes', NULL::text, NULL::timestamptz, NULL::timestamptz, (SELECT id FROM default_org)
		FROM (SELECT id FROM contact_a UNION ALL SELECT id FROM contacts WHERE phone = '+628111111111' AND organization_id = (SELECT id FROM default_org)) c
		WHERE NOT EXISTS (SELECT 1 FROM conversations cv WHERE cv.contact_id = c.id)
		UNION ALL
		SELECT c.id, 'human'::conversation_mode, 'pending_human'::conversation_status, NULL::uuid, NOW() - INTERVAL '10 minutes', 'Needs salary negotiation clarification', NOW() - INTERVAL '10 minutes', NULL::timestamptz, (SELECT id FROM default_org)
		FROM (SELECT id FROM contact_b UNION ALL SELECT id FROM contacts WHERE phone = '+628122222222' AND organization_id = (SELECT id FROM default_org)) c
		WHERE NOT EXISTS (SELECT 1 FROM conversations cv WHERE cv.contact_id = c.id)
		UNION ALL
		SELECT c.id, 'human'::conversation_mode, 'open'::conversation_status, (SELECT id FROM support_agent), NOW() - INTERVAL '15 minutes', NULL::text, NOW() - INTERVAL '40 minutes', NOW() - INTERVAL '35 minutes', (SELECT id FROM default_org)
		FROM (SELECT id FROM resolved_contact UNION ALL SELECT id FROM contacts WHERE phone = '+628133333333' AND organization_id = (SELECT id FROM default_org)) c
		WHERE NOT EXISTS (SELECT 1 FROM conversations cv WHERE cv.contact_id = c.id)
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO messages (conversation_id, external_message_id, sender_type, direction, content_type, text, created_at, organization_id)
		SELECT c.id, 'seed-msg-' || replace(ct.phone, '+', ''), 'customer', 'inbound', 'text',
		       CASE ct.phone
		         WHEN '+628111111111' THEN 'Halo, kapan tim operasional bisa menghubungi saya?'
		         WHEN '+628122222222' THEN 'Apakah ada info harga untuk paket UI/UX?'
		         ELSE 'Terima kasih, saya sudah paham proses layanannya.'
		       END,
		       NOW() - INTERVAL '20 minutes',
		       c.organization_id
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		WHERE NOT EXISTS (
		  SELECT 1 FROM messages m WHERE m.conversation_id = c.id
		)
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		UPDATE conversations c
		SET last_message_id = (
		  SELECT id
		  FROM messages
		  WHERE conversation_id = c.id
		  ORDER BY created_at DESC
		  LIMIT 1
		)
		WHERE c.last_message_id IS NULL
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO conversation_events (conversation_id, event_type, actor_type, payload, organization_id)
		SELECT c.id,
		       CASE
		         WHEN c.status = 'pending_human' THEN 'escalated'::conversation_event_type
		         WHEN c.status = 'resolved' THEN 'conversation_resolved'::conversation_event_type
		         ELSE 'conversation_opened'::conversation_event_type
		       END,
		       'system',
		       jsonb_build_object('seeded', TRUE, 'mode', c.mode, 'status', c.status),
		       c.organization_id
		FROM conversations c
		WHERE NOT EXISTS (
		  SELECT 1 FROM conversation_events e WHERE e.conversation_id = c.id
		)
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO system_status (service_name, status, details)
		VALUES ('app-backend', 'up', '{"service":"app-backend"}')
		ON CONFLICT (service_name) DO UPDATE SET
		  status = EXCLUDED.status,
		  details = EXCLUDED.details,
		  updated_at = NOW()
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO system_status (service_name, status, details)
		VALUES
		  ('wa-gateway', 'disconnected', '{"mock":true,"needsQr":true}'),
		  ('ai-service', 'up', '{"provider":"mock"}'),
		  ('worker', 'up', '{"mode":"heartbeat"}')
		ON CONFLICT (service_name) DO NOTHING
	`)
	return err
}

func seedDemoAgents(ctx context.Context, pool *pgxpool.Pool) error {
	for _, agent := range agents {
		hash, err := bcrypt.GenerateFromPassword([]byte(agent.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO agents (name, username, password_hash, role, platform_role)
			VALUES ($1, $2, $3, $4::agent_role, CASE WHEN $4::text = 'owner' THEN 'platform_owner' ELSE 'none' END)
			ON CONFLICT (username) DO UPDATE SET
			  name = EXCLUDED.name,
			  role = EXCLUDED.role,
			  platform_role = EXCLUDED.platform_role,
			  updated_at = NOW()
		`, agent.Name, agent.Username, string(hash), agent.Role)
		if err != nil {
			return err
		}
	}
	return nil
}

func seedDefaultOrganizationMembership(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		UPDATE agents
		SET current_organization_id = (SELECT id FROM default_org)
		WHERE current_organization_id IS NULL
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		WITH default_org AS (
		  SELECT id FROM organizations WHERE slug = 'default-company'
		)
		INSERT INTO organization_members (organization_id, agent_id, role, status)
		SELECT
		  (SELECT id FROM default_org),
		  id,
		  CASE role::text
		    WHEN 'owner' THEN 'org_owner'
		    WHEN 'super_admin' THEN 'org_admin'
		    WHEN 'admin' THEN 'org_admin'
		    ELSE 'operator'
		  END,
		  CASE WHEN is_active THEN 'active' ELSE 'disabled' END
		FROM agents
		WHERE role::text = 'owner'
		   OR NOT EXISTS (
		     SELECT 1
		     FROM organization_members om
		     JOIN organizations o ON o.id = om.organization_id
		     WHERE om.agent_id = agents.id
		       AND o.slug <> 'default-company'
		       AND om.status = 'active'
		       AND o.status = 'active'
		   )
		ON CONFLICT (organization_id, agent_id) DO UPDATE SET
		  role = EXCLUDED.role,
		  status = EXCLUDED.status,
		  updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM organization_members om
		WHERE om.organization_id = (
		    SELECT id FROM organizations WHERE slug = 'default-company' LIMIT 1
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
		  )
	`)
	return err
}

func seedInitialOwner(ctx context.Context, pool *pgxpool.Pool) error {
	password := strings.TrimSpace(os.Getenv("INITIAL_OWNER_PASSWORD"))
	if password == "" {
		return nil
	}
	username := strings.TrimSpace(os.Getenv("INITIAL_OWNER_USERNAME"))
	if username == "" {
		username = "owner"
	}
	name := strings.TrimSpace(os.Getenv("INITIAL_OWNER_NAME"))
	if name == "" {
		name = "Platform Owner"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO agents (name, username, password_hash, role, platform_role, is_active)
		VALUES ($1, $2, $3, 'owner', 'platform_owner', TRUE)
		ON CONFLICT (username) DO UPDATE SET
		  name = EXCLUDED.name,
		  password_hash = EXCLUDED.password_hash,
		  role = EXCLUDED.role,
		  platform_role = EXCLUDED.platform_role,
		  is_active = TRUE,
		  updated_at = NOW()
	`, name, username, string(hash))
	return err
}

func demoAccountsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_DEMO_ACCOUNTS"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
