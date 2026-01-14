// Package security provides security utilities and validation functions.
package security

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateNetworkName validates network names to prevent command injection.
func ValidateNetworkName(name string) error {
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	// Only allow alphanumeric, hyphens, and underscores
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("invalid network name: must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Limit length
	if len(name) > 255 {
		return fmt.Errorf("network name too long: maximum 255 characters")
	}

	return nil
}

// ValidateNamespaceName validates namespace names to prevent command injection.
func ValidateNamespaceName(name string) error {
	if name == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}

	// Only allow alphanumeric, hyphens, and underscores
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("invalid namespace name: must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Limit length
	if len(name) > 255 {
		return fmt.Errorf("namespace name too long: maximum 255 characters")
	}

	return nil
}

// ValidateIPAddress validates IP addresses.
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	// Simple IPv4 validation
	ipPattern := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipPattern.MatchString(ip) {
		return fmt.Errorf("invalid IP address format")
	}

	// Validate each octet
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		var octet int
		if _, err := fmt.Sscanf(part, "%d", &octet); err != nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
		if octet < 0 || octet > 255 {
			return fmt.Errorf("invalid IP address: octet out of range")
		}
	}

	return nil
}

// ValidateCIDR validates CIDR notation.
func ValidateCIDR(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("CIDR cannot be empty")
	}

	// Simple CIDR validation
	cidrPattern := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	if !cidrPattern.MatchString(cidr) {
		return fmt.Errorf("invalid CIDR format")
	}

	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid CIDR format")
	}

	// Validate IP part
	if err := ValidateIPAddress(parts[0]); err != nil {
		return fmt.Errorf("invalid CIDR: %w", err)
	}

	// Validate prefix length
	var prefix int
	if _, err := fmt.Sscanf(parts[1], "%d", &prefix); err != nil {
		return fmt.Errorf("invalid CIDR prefix: %s", parts[1])
	}
	if prefix < 0 || prefix > 32 {
		return fmt.Errorf("invalid CIDR prefix: must be between 0 and 32")
	}

	return nil
}

// SanitizeString removes potentially dangerous characters from strings.
func SanitizeString(s string) string {
	// Remove control characters and other dangerous chars
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1 // Remove control characters
		}
		// Remove shell metacharacters
		if strings.ContainsRune(";|&$<>`\\\"'", r) {
			return -1
		}
		return r
	}, s)

	return strings.TrimSpace(s)
}

// ValidatePassword validates password strength.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must not exceed 128 characters")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>?]`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character")
	}

	return nil
}

// ValidateUsername validates username format.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}

	if len(username) > 64 {
		return fmt.Errorf("username must not exceed 64 characters")
	}

	// Only allow alphanumeric, underscores, hyphens, and periods
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validPattern.MatchString(username) {
		return fmt.Errorf("invalid username: must contain only alphanumeric characters, periods, hyphens, and underscores")
	}

	// Must start with a letter or number
	if !regexp.MustCompile(`^[a-zA-Z0-9]`).MatchString(username) {
		return fmt.Errorf("invalid username: must start with a letter or number")
	}

	return nil
}

// ValidateEmail validates email format.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Simple email validation
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	if len(email) > 254 {
		return fmt.Errorf("email too long: maximum 254 characters")
	}

	return nil
}
