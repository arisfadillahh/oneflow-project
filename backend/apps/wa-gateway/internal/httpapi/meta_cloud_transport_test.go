package httpapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestValidMetaSignature(t *testing.T) {
	payload := []byte(`{"object":"whatsapp_business_account"}`)
	secret := "meta-app-secret"
	signature := hmac.New(sha256.New, []byte(secret))
	_, _ = signature.Write(payload)

	if !validMetaSignature(payload, "sha256="+hex.EncodeToString(signature.Sum(nil)), secret) {
		t.Fatal("expected Meta signature to be accepted")
	}
	if validMetaSignature([]byte(`{"changed":true}`), "sha256="+hex.EncodeToString(signature.Sum(nil)), secret) {
		t.Fatal("expected changed payload to be rejected")
	}
}

func TestDecryptMetaCredential(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte("business-access-token"), nil)

	value, err := decryptMetaCredential(base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("decrypt token: %v", err)
	}
	if value != "business-access-token" {
		t.Fatalf("unexpected plaintext: %q", value)
	}
}

func TestMetaHistoryDeclinePayload(t *testing.T) {
	var event metaSyncWebhook
	err := json.Unmarshal([]byte(`{"id":"waba-1","event":"history","data":{"metadata":{"phone_number_id":"phone-1"},"history":[{"errors":[{"code":2593109}]}]}}`), &event)
	if err != nil {
		t.Fatalf("parse history event: %v", err)
	}
	if event.Data.Metadata.PhoneNumberID != "phone-1" || metaHistorySyncStatus(event) != "declined" {
		t.Fatalf("unexpected history event parse: %#v", event)
	}
}

func TestMetaBusinessAppEchoPayloadUsesRecipientPhone(t *testing.T) {
	var envelope metaWebhookEnvelope
	err := json.Unmarshal([]byte(`{"entry":[{"changes":[{"field":"smb_message_echoes","value":{"metadata":{"phone_number_id":"business-phone"},"message_echoes":[{"from":"business-phone","to":"628111","id":"echo-1","timestamp":"1710000000","type":"text","text":{"body":"Halo kak"}}]}}]}]}`), &envelope)
	if err != nil {
		t.Fatalf("parse message echo: %v", err)
	}
	echoes := envelope.Entry[0].Changes[0].Value.MessageEchoes
	if len(echoes) != 1 || echoes[0].To != "628111" || echoes[0].Text.Body != "Halo kak" {
		t.Fatalf("unexpected message echo parse: %#v", echoes)
	}
}

func TestValidateMetaTemplateDraftNormalizesNameAndCategory(t *testing.T) {
	draft := metaTemplateDraft{
		Name:     " Status_Update ",
		Category: "utility",
		Language: "id",
		Body:     "Halo kak, pesanan sedang diproses.",
	}
	if err := validateMetaTemplateDraft(&draft); err != nil {
		t.Fatalf("validate template: %v", err)
	}
	if draft.Name != "status_update" || draft.Category != "UTILITY" {
		t.Fatalf("unexpected normalized template: %#v", draft)
	}
}

func TestValidateMetaTemplateDraftRejectsUnsafeName(t *testing.T) {
	draft := metaTemplateDraft{
		Name:     "promo diskon",
		Category: "MARKETING",
		Language: "id",
		Body:     "Promo baru tersedia.",
	}
	if err := validateMetaTemplateDraft(&draft); err == nil {
		t.Fatal("expected invalid template name to be rejected")
	}
}

func TestValidateMetaTemplateSendRequestNormalizesPhone(t *testing.T) {
	request := metaTemplateSendRequest{Phone: "+62 812-3456", Name: "status_update", Language: "id"}
	if err := validateMetaTemplateSendRequest(&request); err != nil {
		t.Fatalf("validate template send: %v", err)
	}
	if request.Phone != "628123456" {
		t.Fatalf("unexpected normalized phone: %q", request.Phone)
	}
}
