package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// parseOndemandServer interprets the -ondemand-server flag, an
// [http|https]://host:port URL for the cache/on-demand asset server. The scheme
// and port are mandatory and a path is rejected (the value is a base that
// signlink.OpenURL appends request fragments to). Returns the normalized
// scheme://host:port.
func parseOndemandServer(arg string) (string, error) {
	if !strings.Contains(arg, "://") {
		return "", fmt.Errorf("ondemand-server %q must include a scheme (http:// or https://)", arg)
	}
	u, err := url.Parse(arg)
	if err != nil {
		return "", fmt.Errorf("parse ondemand-server %q: %w", arg, err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported ondemand-server scheme %q (use http:// or https://)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("ondemand-server %q has no host", arg)
	}
	p := u.Port()
	if p == "" {
		return "", fmt.Errorf("ondemand-server %q must include an explicit port", arg)
	}
	portNum, err := strconv.Atoi(p)
	if err != nil {
		return "", fmt.Errorf("invalid ondemand-server port in %q: %w", arg, err)
	}
	if portNum < 1 || portNum > 65535 {
		return "", fmt.Errorf("ondemand-server port %d out of range (1-65535)", portNum)
	}
	if path := u.EscapedPath(); path != "" && path != "/" {
		return "", fmt.Errorf("ondemand-server %q must not include a path", arg)
	}
	return u.Scheme + "://" + net.JoinHostPort(host, p), nil
}
