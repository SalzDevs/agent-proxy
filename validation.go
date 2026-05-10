package groxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func validateConfig(config Config) error {
	if err := validateAddr(config.Addr); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if config.Timeouts != nil {
		if err := validateTimeouts(*config.Timeouts); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
	}

	if config.MaxBodySize < 0 {
		return fmt.Errorf("invalid config: max body size cannot be negative")
	}

	if err := validateHTTPSInspection(config.HTTPSInspection); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

func validateHTTPSInspection(config *HTTPSInspectionConfig) error {
	if config == nil {
		return nil
	}

	if config.CA == nil {
		return fmt.Errorf("HTTPS inspection CA is required")
	}
	if config.CA.cert == nil || config.CA.key == nil {
		return fmt.Errorf("HTTPS inspection CA is not initialized")
	}
	if !config.CA.cert.IsCA {
		return fmt.Errorf("HTTPS inspection certificate is not a CA")
	}
	if config.Intercept == nil {
		return fmt.Errorf("HTTPS inspection intercept matcher is required")
	}
	if config.CertificateTTL < 0 {
		return fmt.Errorf("HTTPS inspection certificate TTL cannot be negative")
	}

	return nil
}

func validateAddr(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("address is required")
	}

	_, port, err := parseAddr(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	if err := validatePort(port); err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	return nil
}

func parseAddr(addr string) (string, string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return "", "", err
	}

	return host, port, nil
}

func validatePort(port string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port must be numeric: %w", err)
	}

	if portInt < 1 || portInt > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func validateTimeouts(timeouts Timeouts) error {
	if timeouts.Dial < 0 {
		return fmt.Errorf("dial timeout cannot be negative")
	}
	if timeouts.TLSHandshake < 0 {
		return fmt.Errorf("TLS handshake timeout cannot be negative")
	}
	if timeouts.ResponseHeader < 0 {
		return fmt.Errorf("response header timeout cannot be negative")
	}
	if timeouts.IdleConn < 0 {
		return fmt.Errorf("idle connection timeout cannot be negative")
	}
	if timeouts.ReadHeader < 0 {
		return fmt.Errorf("read header timeout cannot be negative")
	}
	if timeouts.Idle < 0 {
		return fmt.Errorf("idle timeout cannot be negative")
	}

	return nil
}
