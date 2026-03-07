package compute

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"

	"github.com/gin-gonic/gin"
)

func startsWithAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// Helper functions for extracting user context.
func (s *Service) getUserIDFromContext(c *gin.Context) uint {
	// Extract user_id set by JWT auth middleware.
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		} else if id, ok := userID.(float64); ok {
			return uint(id)
		}
	}
	// No authenticated user — return 0 to trigger 401 in handlers.
	return 0
}

func (s *Service) getProjectIDFromContext(c *gin.Context) uint {
	if projectID, exists := c.Get("project_id"); exists {
		if id, ok := projectID.(uint); ok {
			return id
		} else if id, ok := projectID.(float64); ok {
			return uint(id)
		}
	}
	return 0
}

// genUUIDv4 generates a random UUIDv4 string without external deps.
func genUUIDv4() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// fallback to a simple pseudo id if rng fails
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	hexs := make([]byte, 36)
	hex.Encode(hexs[0:8], b[0:4])
	hexs[8] = '-'
	hex.Encode(hexs[9:13], b[4:6])
	hexs[13] = '-'
	hex.Encode(hexs[14:18], b[6:8])
	hexs[18] = '-'
	hex.Encode(hexs[19:23], b[8:10])
	hexs[23] = '-'
	hex.Encode(hexs[24:36], b[10:16])
	return string(hexs)
}

// validatedImportURL holds a sanitized URL where the hostname has been replaced
// with its resolved IP address, plus the original Host header for virtual hosts.
type validatedImportURL struct {
	url      url.URL // URL with hostname replaced by resolved IP
	origHost string  // original Host header (hostname:port)
}

// String returns the IP-pinned URL string.
func (v *validatedImportURL) String() string {
	return v.url.String()
}

// validateImportURL validates that rawURL is a safe HTTP(S) URL for image import.
// It rejects non-HTTP schemes and URLs that resolve to private/loopback addresses
// to prevent SSRF attacks. Returns a validatedImportURL with the hostname replaced
// by the resolved IP (preventing DNS rebinding) or an error.
func validateImportURL(rawURL string) (*validatedImportURL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("only http/https allowed")
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host")
	}
	port := parsed.Port()

	// Resolve host to check for private/loopback addresses (SSRF prevention).
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve host: %w", err)
	}
	var resolvedIP string
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return nil, fmt.Errorf("URL resolves to private/loopback address")
		}
		if resolvedIP == "" {
			resolvedIP = ipStr
		}
	}
	if resolvedIP == "" {
		return nil, fmt.Errorf("no valid IP resolved")
	}

	// Pin to resolved IP to prevent DNS rebinding.
	ipHost := resolvedIP
	if port != "" {
		ipHost = net.JoinHostPort(resolvedIP, port)
	}

	return &validatedImportURL{
		url: url.URL{
			Scheme:   parsed.Scheme,
			Host:     ipHost,
			Path:     parsed.Path,
			RawQuery: parsed.RawQuery,
		},
		origHost: parsed.Host,
	}, nil
}
