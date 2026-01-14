package security

import "testing"

func TestValidateNetworkName(t *testing.T) {
	if err := ValidateNetworkName("my-network"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidateNetworkName(""); err == nil {
		t.Error("expected error for empty name")
	}
	if err := ValidateNetworkName("my network!"); err == nil {
		t.Error("expected error for invalid chars")
	}
}

func TestValidateIPAddress(t *testing.T) {
	if err := ValidateIPAddress("192.168.1.1"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidateIPAddress("256.1.1.1"); err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestValidateCIDR(t *testing.T) {
	if err := ValidateCIDR("192.168.1.0/24"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidateCIDR("192.168.1.0/33"); err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("Test@123"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidatePassword("test"); err == nil {
		t.Error("expected error for weak password")
	}
}

func TestValidateUsername(t *testing.T) {
	if err := ValidateUsername("user123"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidateUsername("ab"); err == nil {
		t.Error("expected error for short username")
	}
}

func TestValidateEmail(t *testing.T) {
	if err := ValidateEmail("user@example.com"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := ValidateEmail("invalid"); err == nil {
		t.Error("expected error for invalid email")
	}
}

func TestSanitizeString(t *testing.T) {
	result := SanitizeString("hello; world")
	if result != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result)
	}
}
