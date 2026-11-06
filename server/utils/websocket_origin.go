package utils

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CheckWebSocketOrigin permits same-origin browser WebSockets and clients that
// omit Origin (for example, native MCP/media clients). Forwarded headers are
// deliberately ignored because the HTTP server does not trust proxies.
func CheckWebSocketOrigin(r *http.Request) bool {
	originHeader := strings.TrimSpace(r.Header.Get("Origin"))
	if originHeader == "" {
		return true
	}

	origin, err := url.Parse(originHeader)
	if err != nil || origin.Opaque != "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" {
		return false
	}
	if origin.Scheme != "http" && origin.Scheme != "https" {
		return false
	}
	if origin.Host == "" {
		return false
	}

	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	if origin.Scheme != requestScheme {
		return false
	}

	originHost, originPort, ok := normalizedOriginAuthority(origin.Host, origin.Scheme)
	if !ok {
		return false
	}
	requestHost, requestPort, ok := normalizedOriginAuthority(r.Host, requestScheme)
	if !ok {
		return false
	}

	return strings.EqualFold(originHost, requestHost) && originPort == requestPort
}

func normalizedOriginAuthority(authority, scheme string) (string, string, bool) {
	if authority == "" || strings.ContainsAny(authority, "/\\@") {
		return "", "", false
	}
	for _, char := range authority {
		if char <= 31 || char == 127 {
			return "", "", false
		}
	}

	host := ""
	port := ""
	if strings.HasPrefix(authority, "[") {
		if strings.HasSuffix(authority, "]") {
			host = strings.TrimSuffix(strings.TrimPrefix(authority, "["), "]")
		} else {
			var err error
			host, port, err = net.SplitHostPort(authority)
			if err != nil {
				return "", "", false
			}
		}
	} else if strings.Count(authority, ":") == 0 {
		host = authority
	} else {
		var err error
		host, port, err = net.SplitHostPort(authority)
		if err != nil {
			return "", "", false
		}
	}
	if strings.HasSuffix(authority, ":") {
		return "", "", false
	}

	host = strings.TrimSpace(host)
	// A single trailing dot is the DNS root notation for the same hostname.
	// Browsers preserve it in both Host and Origin, so normalize it instead of
	// rejecting an otherwise same-origin direct-device connection.
	host = strings.TrimSuffix(host, ".")
	if host == "" || strings.HasSuffix(host, ".") {
		return "", "", false
	}
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	}
	if port == "" {
		switch scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return "", "", false
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", "", false
	}

	return host, strconv.Itoa(portNumber), true
}
