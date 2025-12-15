package auth

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	trustedProxies     []*net.IPNet
	trustedProxiesOnce sync.Once
	debugIPLogging     bool
)

// initTrustedProxies initializes the trusted proxy list from environment variable
// TRUSTED_PROXIES should be a comma-separated list of CIDR ranges or IPs
// Example: TRUSTED_PROXIES=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
// Special values:
//   - "private" or "all-private": trust all private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
//   - "*" or "all": trust all IPs (use with caution!)
func initTrustedProxies() {
	trustedProxiesOnce.Do(func() {
		debugIPLogging = os.Getenv("DEBUG_IP") == "true"

		envValue := os.Getenv("TRUSTED_PROXIES")
		if envValue == "" {
			// Default: don't trust any proxies (use RemoteAddr directly)
			log.Printf("TRUSTED_PROXIES not set, will use RemoteAddr directly")
			return
		}

		envValue = strings.TrimSpace(envValue)
		log.Printf("TRUSTED_PROXIES configured: %s", envValue)

		// Handle special values
		switch strings.ToLower(envValue) {
		case "private", "all-private":
			// Trust all private IP ranges - common in container environments
			privateRanges := []string{
				"10.0.0.0/8",
				"172.16.0.0/12",
				"192.168.0.0/16",
				"fc00::/7",    // IPv6 private
				"127.0.0.0/8", // Loopback IPv4
				"::1/128",     // Loopback IPv6
			}
			for _, cidr := range privateRanges {
				_, network, err := net.ParseCIDR(cidr)
				if err == nil {
					trustedProxies = append(trustedProxies, network)
				}
			}
			log.Printf("Trusting private IP ranges: %v", privateRanges)
			return
		case "*", "all":
			// Trust all IPs - equivalent to previous behavior
			// We use 0.0.0.0/0 and ::/0 to match everything
			_, ipv4All, _ := net.ParseCIDR("0.0.0.0/0")
			_, ipv6All, _ := net.ParseCIDR("::/0")
			trustedProxies = append(trustedProxies, ipv4All, ipv6All)
			log.Printf("Trusting ALL IPs (not recommended for production)")
			return
		}

		// Parse comma-separated list of CIDRs or IPs
		for _, entry := range strings.Split(envValue, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}

			// Try parsing as CIDR
			_, network, err := net.ParseCIDR(entry)
			if err == nil {
				trustedProxies = append(trustedProxies, network)
				continue
			}

			// Try parsing as single IP and convert to CIDR
			ip := net.ParseIP(entry)
			if ip != nil {
				var mask net.IPMask
				if ip.To4() != nil {
					mask = net.CIDRMask(32, 32)
				} else {
					mask = net.CIDRMask(128, 128)
				}
				trustedProxies = append(trustedProxies, &net.IPNet{IP: ip, Mask: mask})
			}
		}
		log.Printf("Loaded %d trusted proxy ranges", len(trustedProxies))
	})
}

// isProxyTrusted checks if the given IP is in the trusted proxy list
func isProxyTrusted(ipStr string) bool {
	initTrustedProxies()

	// If no trusted proxies configured, don't trust any forwarded headers
	if len(trustedProxies) == 0 {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, network := range trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// getRemoteIP extracts just the IP from RemoteAddr
func getRemoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return host
}

// GetClientIP extracts the real client IP from a request
// It checks X-Forwarded-For, X-Real-IP headers (if from a trusted proxy),
// and falls back to RemoteAddr
//
// To enable trusting proxy headers, set the TRUSTED_PROXIES environment variable:
//   - TRUSTED_PROXIES=private          - Trust all private IP ranges (recommended for k8s/docker)
//   - TRUSTED_PROXIES=10.42.0.0/16     - Trust specific CIDR range
//   - TRUSTED_PROXIES=10.0.0.1,10.0.0.2 - Trust specific IPs
//   - TRUSTED_PROXIES=*                - Trust all (not recommended for production)
//
// Set DEBUG_IP=true for verbose logging of IP resolution
func GetClientIP(r *http.Request) string {
	initTrustedProxies()

	remoteIP := getRemoteIP(r)
	xff := r.Header.Get("X-Forwarded-For")
	xri := r.Header.Get("X-Real-IP")

	if debugIPLogging {
		log.Printf("[IP Debug] RemoteAddr=%s, X-Forwarded-For=%q, X-Real-IP=%q, ProxyTrusted=%v",
			remoteIP, xff, xri, isProxyTrusted(remoteIP))
	}

	// Only trust forwarded headers if the immediate connection is from a trusted proxy
	if !isProxyTrusted(remoteIP) {
		if debugIPLogging {
			log.Printf("[IP Debug] Proxy not trusted, returning RemoteAddr: %s", remoteIP)
		}
		return remoteIP
	}

	// Check X-Forwarded-For header (may contain multiple IPs)
	if xff != "" {
		// Take the first IP (original client)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if parsed := net.ParseIP(ip); parsed != nil {
				if debugIPLogging {
					log.Printf("[IP Debug] Using X-Forwarded-For: %s", ip)
				}
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri != "" {
		if parsed := net.ParseIP(xri); parsed != nil {
			if debugIPLogging {
				log.Printf("[IP Debug] Using X-Real-IP: %s", xri)
			}
			return xri
		}
	}

	// Fall back to RemoteAddr
	if debugIPLogging {
		log.Printf("[IP Debug] No valid forwarded IP, using RemoteAddr: %s", remoteIP)
	}
	return remoteIP
}

// GetClientIPFromWebSocket extracts the client IP from a WebSocket connection's underlying HTTP request
func GetClientIPFromWebSocket(r *http.Request) string {
	return GetClientIP(r)
}

// NormalizeIP normalizes an IP address to a consistent format
func NormalizeIP(ipStr string) (string, error) {
	// Handle IPv6 zone identifiers
	if idx := strings.Index(ipStr, "%"); idx != -1 {
		ipStr = ipStr[:idx]
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Return the string representation which normalizes the format
	return ip.String(), nil
}

// IsPrivateIP checks if an IP address is in a private/local range
func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 private
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
