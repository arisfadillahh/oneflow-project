package httpapi

import "net/http"

func retiredLegacyEndpoint(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]string{"error": "unofficial WhatsApp has been retired; use Meta Coexistence"})
}
