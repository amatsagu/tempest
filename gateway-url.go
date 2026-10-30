package tempest

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var gatewayURLPattern = regexp.MustCompile(`^(?:wss://(?:[a-z0-9-]+\.)*discord\.gg|wss?://(?:localhost|127\.0\.0\.1|\[::1\]))(?::[0-9]{1,5})?(?:/[^?#\s]*)?(?:\?[^#\s]*)?$`)

func validateGatewayURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid gateway URL: %w", err)
	}

	if u.Scheme != "wss" && u.Scheme != "ws" {
		return "", fmt.Errorf("unsupported gateway URL scheme %q", u.Scheme)
	}

	if u.Host == "" {
		return "", errors.New("gateway URL is missing a host")
	}

	if u.User != nil {
		return "", errors.New("gateway URL must not contain user information")
	}

	if u.Fragment != "" || u.RawFragment != "" {
		return "", errors.New("gateway URL must not contain a fragment")
	}

	host := strings.ToLower(u.Hostname())
	if !isAllowedGatewayHost(host) {
		return "", fmt.Errorf("gateway host %q is not allowed", host)
	}

	if u.Scheme == "ws" && !isLoopbackGatewayHost(host) {
		return "", errors.New("plain ws gateway URLs are only allowed for loopback hosts")
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawFragment = ""
	if u.Path == "" {
		u.Path = "/"
	}

	return u.String(), nil
}

func isAllowedGatewayHost(host string) bool {
	return host == "discord.gg" ||
		strings.HasSuffix(host, ".discord.gg") ||
		isLoopbackGatewayHost(host)
}

func isLoopbackGatewayHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
