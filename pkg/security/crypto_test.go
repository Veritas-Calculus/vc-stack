package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrypto(t *testing.T) {
	// 1. Generate local master key
	keyB64, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey failed: %v", err)
	}

	// Setup environment test
	os.Setenv("VC_MASTER_KEY", keyB64)
	defer os.Unsetenv("VC_MASTER_KEY")

	key, err := GetMasterKey("")
	if err != nil {
		t.Fatalf("GetMasterKey from env failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("Expected key length 32, got %d", len(key))
	}

	// 2. Encrypt and Decrypt Loop
	plainTexts := []string{
		"my_db_password123!",
		"just-some-random-text",
		"another-secret@1",
	}

	for _, pt := range plainTexts {
		enc, err := Encrypt(pt, key)
		if err != nil {
			t.Errorf("Encrypt failed for %s: %v", pt, err)
			continue
		}

		if !strings.HasPrefix(enc, "ENC(") || !strings.HasSuffix(enc, ")") {
			t.Errorf("Expected encrypted string to be wrapped in ENC(), got: %s", enc)
			continue
		}

		dec, err := Decrypt(enc, key)
		if err != nil {
			t.Errorf("Decrypt failed for %s: %v", enc, err)
			continue
		}

		if dec != pt {
			t.Errorf("Expected decrypted string to be '%s', got '%s'", pt, dec)
		}
	}
}

func TestCrypto_PlaintextFallback(t *testing.T) {
	keyB64, _ := GenerateMasterKey()
	os.Setenv("VC_MASTER_KEY", keyB64)
	defer os.Unsetenv("VC_MASTER_KEY")

	key, _ := GetMasterKey("")

	plain := "not-encrypted-string"
	dec, err := Decrypt(plain, key)
	if err != nil {
		t.Fatalf("Decrypt plain failed: %v", err)
	}

	if dec != plain {
		t.Errorf("Expected fallback string to be '%s', got '%s'", plain, dec)
	}
}

func TestGetMasterKey_File(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "master.key")

	keyB64, _ := GenerateMasterKey()
	if err := os.WriteFile(keyFile, []byte(keyB64+"\n"), 0400); err != nil {
		t.Fatalf("Failed to write temp key file: %v", err)
	}

	// Ensure env is empty
	os.Unsetenv("VC_MASTER_KEY")

	key, err := GetMasterKey(keyFile)
	if err != nil {
		t.Fatalf("GetMasterKey from file failed: %v", err)
	}

	if len(key) != 32 {
		t.Fatalf("Expected key length 32, got %d", len(key))
	}
}
