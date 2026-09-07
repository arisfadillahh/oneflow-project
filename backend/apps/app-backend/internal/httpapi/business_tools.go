package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type businessModuleDefinition struct {
	Key              string                 `json:"key"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	Status           string                 `json:"status"`
	NavLabel         string                 `json:"navLabel"`
	Features         []string               `json:"features"`
	Customizable     []string               `json:"customizable"`
	InstalledView    string                 `json:"installedView"`
	ActivationImpact []string               `json:"activationImpact"`
	AIModes          []businessModuleAIMode `json:"aiModes"`
	AIWarning        string                 `json:"aiWarning"`
}

type businessModuleAIMode struct {
	Mode        string `json:"mode"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type businessModuleState struct {
	Enabled bool
	AIMode  string
}

var businessModuleCatalog = []businessModuleDefinition{
	{
		Key:           "prospects",
		Name:          "Prospek & Follow-up",
		Description:   "Kelola calon pelanggan dari chat, tahap follow-up, dan aktivitas closing.",
		Category:      "Sales",
		Status:        "ready",
		NavLabel:      "Follow-up",
		InstalledView: "deals",
		Features: []string{
			"Pipeline prospek dari chat",
			"Tahap follow-up dan owner deal",
			"Aktivitas closing dan catatan follow-up",
		},
		Customizable: []string{
			"Pipeline dan stage per bisnis",
			"Owner dan prioritas prospek",
			"Catatan aktivitas follow-up",
		},
		ActivationImpact: []string{
			"Menu Follow-up muncul di sidebar Alat Terinstall.",
			"Tim bisa menyimpan prospek, stage, owner, prioritas, dan aktivitas follow-up.",
			"Data prospek tetap dikontrol dari dashboard.",
		},
		AIModes: []businessModuleAIMode{
			{Mode: "off", Label: "AI mati", Description: "AI tidak membaca atau membuat data prospek."},
			{Mode: "read", Label: "Baca saja", Description: "AI boleh membaca konteks prospek untuk menjawab follow-up."},
			{Mode: "draft", Label: "Buat draft", Description: "AI boleh membuat prospek baru dari chat customer."},
			{Mode: "action", Label: "Aksi langsung", Description: "AI boleh membuat prospek baru tanpa review manual."},
		},
		AIWarning: "AI bisa mencatat chat sebagai prospek. Pipeline dan stage tetap bisa dikoreksi admin.",
	},
	{
		Key:           "commerce",
		Name:          "Produk, Stok & Pesanan",
		Description:   "Simpan katalog produk, jumlah stok, dan draft pesanan dalam satu alur commerce.",
		Category:      "Commerce",
		Status:        "ready",
		NavLabel:      "Produk & Stok",
		InstalledView: "commerceProducts",
		Features: []string{
			"Input produk, SKU, harga, dan stok",
			"Stok tersedia dan stok reserved",
			"Buat draft pesanan dari produk aktif",
			"Confirm pesanan untuk reserve stok",
		},
		Customizable: []string{
			"Nama produk, deskripsi, SKU, dan harga",
			"Jumlah stok per bisnis",
			"Data customer, kuantitas, catatan, dan status pesanan",
		},
		ActivationImpact: []string{
			"Menu Produk & Stok dan Pesanan muncul di sidebar Alat Terinstall.",
			"Admin bisa mengelola katalog, harga, stok, dan draft pesanan.",
			"Confirm pesanan akan reserve stok; draft pesanan belum mengunci stok.",
		},
		AIModes: []businessModuleAIMode{
			{Mode: "off", Label: "AI mati", Description: "AI tidak membaca produk, stok, atau pesanan."},
			{Mode: "read", Label: "Baca produk", Description: "AI boleh menjawab produk aktif, harga, dan stok tersedia."},
			{Mode: "draft", Label: "Buat draft pesanan", Description: "AI boleh membuat draft pesanan dari chat customer, lalu admin mengecek sebelum confirm."},
			{Mode: "action", Label: "Buat pesanan dari chat", Description: "AI boleh membuat draft pesanan saat produk dan jumlah sudah jelas. Stok tetap baru reserved saat admin confirm."},
		},
		AIWarning: "AI tidak mengubah harga atau stok. Stok baru berubah saat pesanan dikonfirmasi admin.",
	},
	{
		Key:           "booking",
		Name:          "Booking / Jadwal",
		Description:   "Kelola layanan, durasi, jam tersedia, dan appointment customer per bisnis.",
		Category:      "Service",
		Status:        "ready",
		NavLabel:      "Booking",
		InstalledView: "booking",
		Features: []string{
			"Atur layanan booking dan durasi sesi",
			"Simpan jam operasional dan buffer antar jadwal",
			"Buat dan update appointment customer",
		},
		Customizable: []string{
			"Nama layanan, harga, durasi, dan buffer",
			"Hari serta jam layanan tersedia",
			"Timezone dan status layanan",
		},
		ActivationImpact: []string{
			"Menu Booking muncul di sidebar Alat Terinstall.",
			"Admin bisa mengatur layanan, durasi, harga, jam tersedia, dan appointment.",
			"Sistem menolak appointment yang bentrok pada layanan yang sama.",
		},
		AIModes: []businessModuleAIMode{
			{Mode: "off", Label: "AI mati", Description: "AI tidak membaca layanan atau jadwal booking."},
			{Mode: "read", Label: "Baca jadwal", Description: "AI boleh menjawab layanan dan jam tersedia."},
			{Mode: "draft", Label: "Buat draft booking", Description: "AI boleh membuat draft booking dari chat customer, lalu admin mengecek sebelum confirm."},
			{Mode: "action", Label: "Jadwalkan dari chat", Description: "AI boleh menjadwalkan booking saat layanan, nama customer, nomor WhatsApp, tanggal, dan jam sudah jelas."},
		},
		AIWarning: "Mode draft lebih aman untuk operasional awal. Mode aksi langsung akan membuat appointment terjadwal.",
	},
	{
		Key:           "tickets",
		Name:          "Ticketing & Problem Tracker",
		Description:   "Pantau problem customer, status staging, SLA, PIC, dan ringkasan AI dari chat WhatsApp.",
		Category:      "Support",
		Status:        "ready",
		NavLabel:      "Ticketing",
		InstalledView: "tickets",
		Features: []string{
			"Queue tiket dari chat customer",
			"Status problem, severity, SLA, dan PIC",
			"AI triage untuk judul, ringkasan, prioritas, dan next action",
		},
		Customizable: []string{
			"Tipe issue dan severity",
			"PIC, SLA, status, dan komentar internal",
			"Mode AI: baca, draf triage, atau bantu action operasional",
		},
		ActivationImpact: []string{
			"Menu Ticketing muncul di sidebar Alat Terinstall.",
			"Admin bisa membuat tiket dari chat, kontak, atau input manual.",
			"AI triage memakai kredit sesuai token yang dipakai; action manual tidak memakai kredit Oneflow.",
		},
		AIModes: []businessModuleAIMode{
			{Mode: "off", Label: "AI mati", Description: "AI tidak membaca atau membuat triage tiket."},
			{Mode: "read", Label: "Baca tiket", Description: "AI boleh membaca konteks tiket untuk jawaban dan ringkasan."},
			{Mode: "draft", Label: "Buat draf triage", Description: "AI boleh menyiapkan judul, tipe issue, prioritas, severity, dan next action untuk dicek admin."},
			{Mode: "action", Label: "Bantu update tiket", Description: "AI boleh membantu update status saat aturan action sudah jelas; admin tetap bisa koreksi."},
		},
		AIWarning: "AI triage tiket akan memotong kredit minimal 1 credit per action melalui billing backend. Follow-up WhatsApp official tetap mengikuti aturan window Meta 24 jam.",
	},
	{
		Key:           "payments",
		Name:          "Payment",
		Description:   "Disiapkan untuk konfigurasi pembayaran per bisnis. Integrasi akan dibahas terpisah.",
		Category:      "Shared",
		Status:        "planned",
		NavLabel:      "Payment",
		InstalledView: "",
		Features: []string{
			"Konfigurasi payment provider per bisnis",
			"Payment link setelah order matang",
			"Webhook sukses bayar untuk konfirmasi pesanan",
		},
		Customizable: []string{
			"Provider pembayaran per bisnis",
			"Instruksi pembayaran",
			"Status dan callback pembayaran",
		},
		ActivationImpact: []string{
			"Payment belum bisa diinstall karena masih disiapkan.",
			"Nanti fitur ini akan mengatur provider pembayaran, instruksi bayar, dan callback sukses bayar.",
			"Status lunas harus berasal dari sistem pembayaran, bukan klaim chat customer.",
		},
		AIModes: []businessModuleAIMode{
			{Mode: "off", Label: "AI mati", Description: "AI tidak memakai data pembayaran."},
			{Mode: "read", Label: "Baca instruksi", Description: "AI boleh menjelaskan instruksi pembayaran saat fitur tersedia."},
		},
		AIWarning: "AI tidak boleh refund, mengubah nominal, atau menandai lunas tanpa bukti provider.",
	},
}

func businessModuleDefinitionByKey(key string) (businessModuleDefinition, bool) {
	for _, item := range businessModuleCatalog {
		if item.Key == key {
			return item, true
		}
	}
	return businessModuleDefinition{}, false
}

func canManageBusinessModules(role string) bool {
	return role == "owner" || role == "super_admin" || role == "admin"
}

func (s *Server) handleBusinessTools(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeBusinessTools(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBusinessToolRoutes(w http.ResponseWriter, r *http.Request) {
	routeParts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/business-tools/"), "/"), "/")
	moduleKey := ""
	action := ""
	if len(routeParts) > 0 {
		moduleKey = routeParts[0]
	}
	if len(routeParts) > 1 {
		action = routeParts[1]
	}
	definition, ok := businessModuleDefinitionByKey(moduleKey)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "business tool not found"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		if action != "install" {
			http.NotFound(w, r)
			return
		}
		if !canManageBusinessModules(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage business tools"})
			return
		}
		if definition.Status != "ready" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "business tool is not ready yet"})
			return
		}
		if err := s.setBusinessModuleEnabled(r.Context(), moduleKey, true); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "business_tool.install", "organization_module", "", map[string]any{
			"moduleKey": moduleKey,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeBusinessTools(w, r)
	case http.MethodPatch:
		if action != "" {
			http.NotFound(w, r)
			return
		}
		if !canManageBusinessModules(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage business tools"})
			return
		}
		var req struct {
			Enabled *bool   `json:"enabled"`
			AIMode  *string `json:"aiMode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.Enabled == nil && req.AIMode == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nothing to update"})
			return
		}
		if req.Enabled != nil && *req.Enabled && definition.Status != "ready" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "business tool is not ready yet"})
			return
		}
		enabledValue := false
		if req.Enabled != nil {
			enabledValue = *req.Enabled
			if err := s.setBusinessModuleEnabled(r.Context(), moduleKey, enabledValue); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if !enabledValue && req.AIMode == nil {
				off := "off"
				req.AIMode = &off
			}
		}
		if req.AIMode != nil {
			aiMode := normalizeBusinessModuleAIMode(*req.AIMode)
			if aiMode == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid AI mode"})
				return
			}
			if aiMode != "off" {
				installed, err := s.businessModuleEnabled(r.Context(), moduleKey)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if req.Enabled != nil {
					installed = enabledValue
				}
				if !installed {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "install business tool before enabling AI"})
					return
				}
				if definition.Status != "ready" {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "business tool is not ready yet"})
					return
				}
			}
			if err := s.setBusinessModuleAIMode(r.Context(), moduleKey, aiMode); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		if err := s.insertAuditLog(r.Context(), "business_tool.toggle", "organization_module", "", map[string]any{
			"moduleKey": moduleKey,
			"enabled":   req.Enabled,
			"aiMode":    req.AIMode,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeBusinessTools(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) writeBusinessTools(w http.ResponseWriter, r *http.Request) {
	states, err := s.businessModulesStateMap(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	items := make([]map[string]any, 0, len(businessModuleCatalog))
	for _, item := range businessModuleCatalog {
		state := states[item.Key]
		installed := state.Enabled
		aiMode := normalizeBusinessModuleAIMode(state.AIMode)
		if aiMode == "" {
			aiMode = "off"
		}
		items = append(items, map[string]any{
			"key":              item.Key,
			"name":             item.Name,
			"description":      item.Description,
			"category":         item.Category,
			"status":           item.Status,
			"navLabel":         item.NavLabel,
			"features":         item.Features,
			"customizable":     item.Customizable,
			"installedView":    item.InstalledView,
			"activationImpact": item.ActivationImpact,
			"aiModes":          item.AIModes,
			"aiWarning":        item.AIWarning,
			"aiMode":           aiMode,
			"aiEnabled":        installed && aiMode != "off",
			"installed":        installed,
			"enabled":          installed,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"canManage": canManageBusinessModules(s.role(r.Context())),
	})
}

func (s *Server) businessModulesEnabledMap(ctx context.Context) (map[string]bool, error) {
	states, err := s.businessModulesStateMap(ctx)
	if err != nil {
		return nil, err
	}
	items := map[string]bool{}
	for _, item := range businessModuleCatalog {
		items[item.Key] = states[item.Key].Enabled
	}
	return items, nil
}

func (s *Server) businessModulesStateMap(ctx context.Context) (map[string]businessModuleState, error) {
	items := map[string]businessModuleState{}
	for _, item := range businessModuleCatalog {
		items[item.Key] = businessModuleState{Enabled: false, AIMode: "off"}
	}
	rows, err := s.db.Query(ctx, `
		SELECT module_key, is_enabled, COALESCE(ai_mode, 'off')
		FROM organization_modules
		WHERE organization_id = $1
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, aiMode string
		var enabled bool
		if err := rows.Scan(&key, &enabled, &aiMode); err != nil {
			return nil, err
		}
		if _, ok := items[key]; ok {
			items[key] = businessModuleState{Enabled: enabled, AIMode: normalizeBusinessModuleAIMode(aiMode)}
		}
	}
	return items, rows.Err()
}

func normalizeBusinessModuleAIMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "off", "disabled":
		return "off"
	case "read", "readonly", "read_only":
		return "read"
	case "draft", "assist":
		return "draft"
	case "action", "write", "auto":
		return "action"
	default:
		return ""
	}
}

func aiModeRank(mode string) int {
	switch normalizeBusinessModuleAIMode(mode) {
	case "read":
		return 1
	case "draft":
		return 2
	case "action":
		return 3
	default:
		return 0
	}
}

func (s *Server) businessModuleAIAllowed(ctx context.Context, moduleKey, minimumMode string) (bool, error) {
	if _, ok := businessModuleDefinitionByKey(moduleKey); !ok {
		return false, nil
	}
	var enabled bool
	var aiMode string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(is_enabled, FALSE), COALESCE(ai_mode, 'off')
		FROM organization_modules
		WHERE organization_id = $1 AND module_key = $2
	`, s.organizationID(ctx), moduleKey).Scan(&enabled, &aiMode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return enabled && aiModeRank(aiMode) >= aiModeRank(minimumMode), nil
}

func (s *Server) businessModuleEnabled(ctx context.Context, moduleKey string) (bool, error) {
	if _, ok := businessModuleDefinitionByKey(moduleKey); !ok {
		return false, nil
	}
	var enabled bool
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(is_enabled, FALSE)
		FROM organization_modules
		WHERE organization_id = $1 AND module_key = $2
	`, s.organizationID(ctx), moduleKey).Scan(&enabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

func (s *Server) requireBusinessModule(w http.ResponseWriter, r *http.Request, moduleKey string) bool {
	enabled, err := s.businessModuleEnabled(r.Context(), moduleKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return false
	}
	if !enabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "business tool is not active"})
		return false
	}
	return true
}

func (s *Server) setBusinessModuleEnabled(ctx context.Context, moduleKey string, enabled bool) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx, `
		INSERT INTO organization_modules (
		  organization_id,
		  module_key,
		  is_enabled,
		  enabled_at,
		  enabled_by,
		  disabled_at,
		  disabled_by,
		  updated_at
		)
		VALUES (
		  $1,
		  $2,
		  $3,
		  CASE WHEN $3 THEN $4::timestamptz ELSE NULL::timestamptz END,
		  CASE WHEN $3 THEN NULLIF($5, '')::uuid ELSE NULL END,
		  CASE WHEN $3 THEN NULL::timestamptz ELSE $4::timestamptz END,
		  CASE WHEN $3 THEN NULL ELSE NULLIF($5, '')::uuid END,
		  $4
		)
		ON CONFLICT (organization_id, module_key) DO UPDATE
		SET is_enabled = EXCLUDED.is_enabled,
		    enabled_at = CASE WHEN EXCLUDED.is_enabled THEN COALESCE(organization_modules.enabled_at, EXCLUDED.enabled_at) ELSE organization_modules.enabled_at END,
		    enabled_by = CASE WHEN EXCLUDED.is_enabled THEN COALESCE(organization_modules.enabled_by, EXCLUDED.enabled_by) ELSE organization_modules.enabled_by END,
		    disabled_at = CASE WHEN EXCLUDED.is_enabled THEN NULL ELSE EXCLUDED.disabled_at END,
		    disabled_by = CASE WHEN EXCLUDED.is_enabled THEN NULL ELSE EXCLUDED.disabled_by END,
		    ai_mode = CASE WHEN EXCLUDED.is_enabled THEN organization_modules.ai_mode ELSE 'off' END,
		    updated_at = EXCLUDED.updated_at
	`, s.organizationID(ctx), moduleKey, enabled, now, s.agentID(ctx))
	return err
}

func (s *Server) setBusinessModuleAIMode(ctx context.Context, moduleKey, aiMode string) error {
	normalized := normalizeBusinessModuleAIMode(aiMode)
	if normalized == "" {
		return nil
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx, `
		INSERT INTO organization_modules (
		  organization_id,
		  module_key,
		  is_enabled,
		  ai_mode,
		  ai_updated_at,
		  ai_updated_by,
		  updated_at
		)
		VALUES ($1, $2, FALSE, $3, $4, NULLIF($5, '')::uuid, $4)
		ON CONFLICT (organization_id, module_key) DO UPDATE
		SET ai_mode = EXCLUDED.ai_mode,
		    ai_updated_at = EXCLUDED.ai_updated_at,
		    ai_updated_by = EXCLUDED.ai_updated_by,
		    updated_at = EXCLUDED.updated_at
	`, s.organizationID(ctx), moduleKey, normalized, now, s.agentID(ctx))
	return err
}
