package repository

import (
	"strings"
	"testing"
)

func TestSanitizeAuditJSONRedactsNestedSecrets(t *testing.T) {
	raw := `{"password":"plain","profile":{"access_token":"abc","name":"operator"},"items":[{"client-secret":"xyz"}]}`
	got := sanitizeAuditJSON(raw)
	for _, secret := range []string{"plain", "abc", "xyz"} {
		if strings.Contains(got, secret) {
			t.Fatalf("audit payload leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "operator") || strings.Count(got, "[REDACTED]") != 3 {
		t.Fatalf("unexpected sanitized payload: %s", got)
	}
}

func TestSanitizeAuditJSONRejectsInvalidPayload(t *testing.T) {
	if got := sanitizeAuditJSON("token=secret"); strings.Contains(got, "secret") || !strings.Contains(got, "invalid_json") {
		t.Fatalf("invalid audit payload was not fail-closed: %s", got)
	}
}
