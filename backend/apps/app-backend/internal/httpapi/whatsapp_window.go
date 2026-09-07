package httpapi

import (
	"math"
	"time"
)

const whatsappCustomerServiceWindow = 24 * time.Hour

type whatsappCustomerWindowInfo struct {
	Status                string     `json:"status"`
	LastCustomerMessageAt *time.Time `json:"lastCustomerMessageAt,omitempty"`
	ExpiresAt             *time.Time `json:"expiresAt,omitempty"`
	HoursRemaining        float64    `json:"hoursRemaining"`
	RequiresTemplate      bool       `json:"requiresTemplate"`
	MetaChargeExpected    bool       `json:"metaChargeExpected"`
	Guidance              string     `json:"guidance"`
}

func newWhatsAppCustomerWindowInfo(lastCustomerMessageAt *time.Time, now time.Time) whatsappCustomerWindowInfo {
	if lastCustomerMessageAt == nil || lastCustomerMessageAt.IsZero() {
		return whatsappCustomerWindowInfo{
			Status:             "unknown",
			RequiresTemplate:   true,
			MetaChargeExpected: true,
			Guidance:           "Belum ada inbound customer yang tercatat. Untuk WhatsApp official, kirim approved template sebelum free-form.",
		}
	}
	expiresAt := lastCustomerMessageAt.Add(whatsappCustomerServiceWindow)
	if now.Before(expiresAt) {
		remaining := expiresAt.Sub(now).Hours()
		return whatsappCustomerWindowInfo{
			Status:                "open",
			LastCustomerMessageAt: lastCustomerMessageAt,
			ExpiresAt:             &expiresAt,
			HoursRemaining:        math.Round(remaining*10) / 10,
			RequiresTemplate:      false,
			MetaChargeExpected:    false,
			Guidance:              "Masih di dalam 24 jam sejak pesan customer terakhir. Free-form follow-up boleh dikirim.",
		}
	}
	return whatsappCustomerWindowInfo{
		Status:                "expired",
		LastCustomerMessageAt: lastCustomerMessageAt,
		ExpiresAt:             &expiresAt,
		RequiresTemplate:      true,
		MetaChargeExpected:    true,
		Guidance:              "Di atas 24 jam sejak pesan customer terakhir. WhatsApp official wajib memakai approved template; template biasanya mengikuti biaya kategori Meta/BSP.",
	}
}

func whatsappCustomerWindowAllowsFreeform(lastCustomerMessageAt *time.Time, now time.Time) bool {
	return newWhatsAppCustomerWindowInfo(lastCustomerMessageAt, now).Status == "open"
}
