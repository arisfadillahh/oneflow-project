package httpapi

import "testing"

func TestOfficialOnlyProvider(t *testing.T) {
	for _, input := range []string{"", "meta_cloud", " meta_cloud "} {
		provider, err := officialWhatsAppProvider(input)
		if err != nil || provider != "meta_cloud" {
			t.Fatalf("official provider rejected: %q", input)
		}
	}
	for _, input := range []string{"whatsmeow", "mock", "real", "unknown"} {
		if _, err := officialWhatsAppProvider(input); err == nil {
			t.Fatalf("legacy provider accepted: %q", input)
		}
	}
}
