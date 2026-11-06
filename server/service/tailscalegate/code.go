package tailscalegate

import "errors"

// LoginCodePath is the single Unix-socket path that carries every Tailscale
// login gate interaction. Query parameters select the operation; routing on
// this constant is added with the handler.
const LoginCodePath = "/api/tailscale/login/code"

const (
	offCode               = "%off%"
	onlineDirectivePrefix = "%online%"
)

// errInvalidOnlineDirective is a fixed, non-sensitive error returned when the
// supplied peer values cannot form a valid pairing directive.
var errInvalidOnlineDirective = errors.New("tailscalegate: invalid online directive")

// OffCode returns the compatibility "already off" login code. It is the
// response for a code request before initialization and for ?disable.
func OffCode() string {
	return offCode
}

// PlainCode returns raw filtered to printable ASCII (0x20..0x7E).
//
// The documented contract has no font ladder and no length truncation for an
// ordinary verification code, so the surviving bytes keep their original order
// and count; an empty result stays empty.
func PlainCode(raw string) string {
	return printableASCII(raw, 0)
}

// OnlineDirective builds the "%online%<name>%<ip>%" pairing directive.
//
// The client treats any malformed directive as a parse failure and does not
// fall back to displaying it as a plain code, so neither field may be empty,
// contain a byte outside printable ASCII, or contain '%'. When the inputs are
// invalid this returns a fixed non-sensitive error and no value: the caller
// must never emit an invalid directive.
func OnlineDirective(name, ip string) (string, error) {
	if !isDirectiveField(name) || !isDirectiveField(ip) {
		return "", errInvalidOnlineDirective
	}
	return onlineDirectivePrefix + name + "%" + ip + "%", nil
}

// isDirectiveField reports whether value can appear between the '%' separators
// of an online directive.
func isDirectiveField(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b < 0x20 || b > 0x7E || b == '%' {
			return false
		}
	}
	return true
}
