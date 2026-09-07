package httpapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type organizationPlanLimits struct {
	PlanName            string     `json:"planName"`
	PlanKey             string     `json:"planKey"`
	BillingPeriod       string     `json:"billingPeriod"`
	MaxWhatsAppSessions int        `json:"maxWhatsAppSessions"`
	MaxAIAgents         int        `json:"maxAiAgents"`
	MaxHumanUsers       int        `json:"maxHumanUsers"`
	ActiveUntil         *time.Time `json:"activeUntil,omitempty"`
}

func trialPlanLimits() organizationPlanLimits {
	return organizationPlanLimits{
		PlanName:            "Trial",
		PlanKey:             "trial",
		MaxWhatsAppSessions: 0,
		MaxAIAgents:         1,
		MaxHumanUsers:       1,
	}
}

func planPackageRank(planKey string) int {
	switch strings.ToLower(strings.TrimSpace(planKey)) {
	case "starter":
		return 1
	case "growth":
		return 2
	case "business":
		return 3
	default:
		return 0
	}
}

func (s *Server) activeOrganizationPlanLimits(ctx context.Context) (organizationPlanLimits, error) {
	limits := trialPlanLimits()
	err := s.db.QueryRow(ctx, `
		SELECT
		  COALESCE(NULLIF(cp.name, ''), 'Trial'),
		  COALESCE(NULLIF(cp.plan_key, ''), 'custom'),
		  COALESCE(p.billing_period, cp.billing_period, 'monthly'),
		  COALESCE(cp.max_whatsapp_sessions, 1),
		  COALESCE(cp.max_ai_agents, 1),
		  COALESCE(cp.max_human_users, 1),
		  p.active_until
		FROM credit_purchases p
		JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
		WHERE p.organization_id = $1
		  AND p.payment_status = 'confirmed'
		  AND COALESCE(p.billing_period, cp.billing_period, 'monthly') IN ('monthly', 'annual')
		  AND (p.active_until IS NULL OR p.active_until > NOW())
		ORDER BY CASE LOWER(COALESCE(cp.plan_key, ''))
		  WHEN 'business' THEN 3
		  WHEN 'growth' THEN 2
		  WHEN 'starter' THEN 1
		  ELSE 0
		END DESC, COALESCE(p.active_from, p.confirmed_at, p.created_at) DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&limits.PlanName, &limits.PlanKey, &limits.BillingPeriod, &limits.MaxWhatsAppSessions, &limits.MaxAIAgents, &limits.MaxHumanUsers, &limits.ActiveUntil)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return limits, nil
		}
		return organizationPlanLimits{}, err
	}
	return limits, nil
}

func (s *Server) ensurePlanCapacity(ctx context.Context, resource string) error {
	limits, err := s.activeOrganizationPlanLimits(ctx)
	if err != nil {
		return err
	}
	var current, max int
	var label string
	switch resource {
	case "whatsapp":
		max = limits.MaxWhatsAppSessions
		label = "WhatsApp number"
		err = s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM whatsapp_sessions
			WHERE organization_id = $1 AND deleted_at IS NULL
			  AND provider = 'meta_cloud'
		`, s.organizationID(ctx)).Scan(&current)
	case "ai_agent":
		max = limits.MaxAIAgents
		label = "AI agent"
		err = s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM ai_agents
			WHERE organization_id = $1 AND is_active = TRUE
		`, s.organizationID(ctx)).Scan(&current)
	case "human_user":
		max = limits.MaxHumanUsers
		label = "human user"
		err = s.db.QueryRow(ctx, `
			SELECT
			  (SELECT COUNT(*) FROM organization_members WHERE organization_id = $1 AND status = 'active') +
			  (SELECT COUNT(*) FROM organization_invites WHERE organization_id = $1 AND status = 'pending' AND expires_at > NOW())
		`, s.organizationID(ctx)).Scan(&current)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	if max <= 0 {
		return fmt.Errorf("%s is not included in %s plan. Upgrade package to add it.", label, limits.PlanName)
	}
	if current >= max {
		return fmt.Errorf("%s limit reached for %s plan (%d/%d). Upgrade package to add more.", label, limits.PlanName, current, max)
	}
	return nil
}

func (s *Server) ensureWhatsAppPlanAccess(ctx context.Context) error {
	limits, err := s.activeOrganizationPlanLimits(ctx)
	if err != nil {
		return err
	}
	if limits.MaxWhatsAppSessions <= 0 {
		return fmt.Errorf("WhatsApp connection is available on paid packages only. Use Playground during trial or upgrade package to connect WhatsApp.")
	}
	return nil
}
