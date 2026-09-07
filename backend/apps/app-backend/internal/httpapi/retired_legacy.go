package httpapi

import "net/http"

func retiredLegacyEndpoint(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]string{"error": "legacy WhatsApp group binding has been retired; use dashboard Inbox"})
}
