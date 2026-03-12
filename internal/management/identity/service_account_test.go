package identity

import (
	"testing"
	"time"
)

// ──────────────────────────────────────────────────────────────────────
// Service Account Unit Tests
// ──────────────────────────────────────────────────────────────────────

func TestCreateServiceAccount(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, err := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{
		Name:        "ci-pipeline",
		Description: "Service account for CI/CD pipeline",
	})
	if err != nil {
		t.Fatalf("CreateServiceAccount error: %v", err)
	}

	if resp.AccessKeyID == "" {
		t.Error("expected non-empty AccessKeyID")
	}
	if resp.SecretKey == "" {
		t.Error("expected non-empty SecretKey (returned only on create)")
	}
	if resp.ServiceAccount.Name != "ci-pipeline" {
		t.Errorf("expected name 'ci-pipeline', got %q", resp.ServiceAccount.Name)
	}
	if !resp.ServiceAccount.IsActive {
		t.Error("expected service account to be active by default")
	}
	if resp.ServiceAccount.ID == 0 {
		t.Error("expected non-zero ID after creation")
	}
	// AccessKeyID format: VC-AKIA-{16 hex}
	if len(resp.AccessKeyID) != 24 {
		t.Errorf("expected AccessKeyID length 24, got %d (%q)", len(resp.AccessKeyID), resp.AccessKeyID)
	}
}

func TestCreateServiceAccountWithExpiry(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, err := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{
		Name:      "temp-token",
		ExpiresIn: "24h",
	})
	if err != nil {
		t.Fatalf("CreateServiceAccount error: %v", err)
	}

	if resp.ServiceAccount.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}

	// Should expire approximately 24h from now.
	expectedExpiry := time.Now().Add(24 * time.Hour)
	diff := resp.ServiceAccount.ExpiresAt.Sub(expectedExpiry)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("expected expiry ~24h from now, got diff=%s", diff)
	}
}

func TestCreateServiceAccountInvalidExpiry(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{
		Name:      "bad-expiry",
		ExpiresIn: "not-a-duration",
	})
	if err == nil {
		t.Error("expected error for invalid expires_in, got nil")
	}
}

func TestListServiceAccounts(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create two service accounts.
	for _, name := range []string{"sa-one", "sa-two"} {
		_, err := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{Name: name})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	accounts, err := svc.ListServiceAccounts()
	if err != nil {
		t.Fatalf("ListServiceAccounts error: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 service accounts, got %d", len(accounts))
	}
}

func TestGetServiceAccount(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, _ := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{Name: "get-test"})

	sa, err := svc.GetServiceAccount(resp.ServiceAccount.ID)
	if err != nil {
		t.Fatalf("GetServiceAccount error: %v", err)
	}
	if sa.Name != "get-test" {
		t.Errorf("expected name 'get-test', got %q", sa.Name)
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, _ := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{Name: "to-delete"})

	if err := svc.DeleteServiceAccount(resp.ServiceAccount.ID); err != nil {
		t.Fatalf("DeleteServiceAccount error: %v", err)
	}

	// Should not be found after deletion.
	_, err := svc.GetServiceAccount(resp.ServiceAccount.ID)
	if err == nil {
		t.Error("expected error after deleting service account, but got nil")
	}
}

func TestRotateServiceAccountKey(t *testing.T) {
	svc, _ := setupTestService(t)

	original, _ := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{Name: "rotate-test"})
	oldKey := original.AccessKeyID

	rotated, err := svc.RotateServiceAccountKey(original.ServiceAccount.ID)
	if err != nil {
		t.Fatalf("RotateServiceAccountKey error: %v", err)
	}

	if rotated.AccessKeyID == oldKey {
		t.Error("expected new AccessKeyID to differ from the old one")
	}
	if rotated.SecretKey == "" {
		t.Error("expected non-empty new SecretKey")
	}
	if rotated.ServiceAccount.ID != original.ServiceAccount.ID {
		t.Error("expected same SA ID after rotation")
	}
}

func TestToggleServiceAccountStatus(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, _ := svc.CreateServiceAccount(1, &CreateServiceAccountRequest{Name: "toggle-test"})

	// Deactivate.
	if err := svc.ToggleServiceAccountStatus(resp.ServiceAccount.ID, false); err != nil {
		t.Fatalf("toggle off: %v", err)
	}

	sa, _ := svc.GetServiceAccount(resp.ServiceAccount.ID)
	if sa.IsActive {
		t.Error("expected service account to be inactive after toggle off")
	}

	// Reactivate.
	if err := svc.ToggleServiceAccountStatus(resp.ServiceAccount.ID, true); err != nil {
		t.Fatalf("toggle on: %v", err)
	}

	sa, _ = svc.GetServiceAccount(resp.ServiceAccount.ID)
	if !sa.IsActive {
		t.Error("expected service account to be active after toggle on")
	}
}

func TestServiceAccountIsExpired(t *testing.T) {
	// Not expired: nil ExpiresAt.
	sa1 := &ServiceAccount{}
	if sa1.IsExpired() {
		t.Error("nil ExpiresAt should not be expired")
	}

	// Not expired: future date.
	future := time.Now().Add(1 * time.Hour)
	sa2 := &ServiceAccount{ExpiresAt: &future}
	if sa2.IsExpired() {
		t.Error("future ExpiresAt should not be expired")
	}

	// Expired: past date.
	past := time.Now().Add(-1 * time.Hour)
	sa3 := &ServiceAccount{ExpiresAt: &past}
	if !sa3.IsExpired() {
		t.Error("past ExpiresAt should be expired")
	}
}

func TestComputeHMAC(t *testing.T) {
	key := "test-secret-key"
	akid := "VC-AKIA-0123456789abcdef"
	ts := "1709000000"
	method := "GET"
	path := "/api/v1/instances"

	sig1 := computeHMAC(key, akid, ts, method, path)
	sig2 := computeHMAC(key, akid, ts, method, path)

	if sig1 == "" {
		t.Error("expected non-empty HMAC signature")
	}
	if sig1 != sig2 {
		t.Error("expected same HMAC for identical inputs")
	}

	// Different key → different signature.
	sig3 := computeHMAC("different-key", akid, ts, method, path)
	if sig1 == sig3 {
		t.Error("expected different HMAC for different keys")
	}

	// Different path → different signature.
	sig4 := computeHMAC(key, akid, ts, method, "/api/v1/volumes")
	if sig1 == sig4 {
		t.Error("expected different HMAC for different paths")
	}
}

func TestParseHMACParams(t *testing.T) {
	header := "VC-HMAC-SHA256 AccessKeyId=VC-AKIA-abc123, Timestamp=1709000000, Signature=deadbeef"
	params := parseHMACParams(header)

	if params["AccessKeyId"] != "VC-AKIA-abc123" {
		t.Errorf("expected AccessKeyId 'VC-AKIA-abc123', got %q", params["AccessKeyId"])
	}
	if params["Timestamp"] != "1709000000" {
		t.Errorf("expected Timestamp '1709000000', got %q", params["Timestamp"])
	}
	if params["Signature"] != "deadbeef" {
		t.Errorf("expected Signature 'deadbeef', got %q", params["Signature"])
	}
}

func TestParseHMACParams_Invalid(t *testing.T) {
	// Empty header.
	params := parseHMACParams("")
	if len(params) != 0 {
		t.Errorf("expected 0 params for empty header, got %d", len(params))
	}

	// Wrong prefix.
	params = parseHMACParams("Bearer token123")
	if len(params) != 0 {
		t.Errorf("expected 0 params for Bearer header, got %d", len(params))
	}
}

func TestGenerateAccessKeyID(t *testing.T) {
	id1, err := generateAccessKeyID()
	if err != nil {
		t.Fatalf("generateAccessKeyID error: %v", err)
	}
	id2, _ := generateAccessKeyID()

	// Should start with VC-AKIA-.
	if len(id1) < 8 || id1[:8] != "VC-AKIA-" {
		t.Errorf("expected VC-AKIA- prefix, got %q", id1)
	}
	// Should be unique.
	if id1 == id2 {
		t.Error("expected unique access key IDs")
	}
}

func TestGenerateSecretKey(t *testing.T) {
	key1, err := generateSecretKey()
	if err != nil {
		t.Fatalf("generateSecretKey error: %v", err)
	}
	key2, _ := generateSecretKey()

	if len(key1) == 0 {
		t.Error("expected non-empty secret key")
	}
	if key1 == key2 {
		t.Error("expected unique secret keys")
	}
}
