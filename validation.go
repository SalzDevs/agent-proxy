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
