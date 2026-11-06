package tailscalegate

import (
	"net/http"
	"net/url"
	"strings"
)

// route identifies which behavior LoginCodePath should run for a request.
type route int

const (
	// routeUnknown marks a rejected request. The decision carries the HTTP
	// status to answer with and no behavior runs.
	routeUnknown route = iota
	// routeStart triggers server-side initialization.
	routeStart
	// routeExit triggers server-side deinitialization. It is only accepted
	// once the gate is confirmed ready (start && match).
	routeExit
	// routeDisable ends the client initialization session.
	routeDisable
	// routeHeartbeat returns the global state snapshot.
	routeHeartbeat
	// routeCode returns the login code or a reserved directive.
	routeCode
)

// decision is the classification of a LoginCodePath request.
//
// A dispatched request has a non-unknown Route and Status 0. A rejected
// request has routeUnknown and the HTTP status to answer with: 400 for a
// non-GET method and 404 for an unknown path.
type decision struct {
	Route  route
	Status int
}

// dispatch classifies a login-code request by method, path, and query flags.
//
// The order matches the reference server: the method is rejected before the
// path is inspected, and the surviving query flags are tried in the fixed
// order start -> exit -> disable -> heartbeat. The first flag that matches
// wins, so a request may carry several query parameters; only the
// highest-priority one is honored.
//
// dispatch is a pure function: it performs no I/O, reads no configuration,
// and never writes a response. The feature-manifest gate is applied by the
// caller because reading configuration is not part of request
// classification.
func dispatch(r *http.Request) decision {
	if r == nil {
		return decision{Route: routeUnknown, Status: http.StatusBadRequest}
	}
	if r.Method != http.MethodGet {
		return decision{Route: routeUnknown, Status: http.StatusBadRequest}
	}
	if r.URL == nil || r.URL.Path != LoginCodePath {
		return decision{Route: routeUnknown, Status: http.StatusNotFound}
	}

	query := r.URL.Query()
	switch {
	case queryFlag(query, "start"):
		return decision{Route: routeStart}
	case queryFlag(query, "exit"):
		return decision{Route: routeExit}
	case queryFlag(query, "disable"):
		return decision{Route: routeDisable}
	case queryFlag(query, "heartbeat"):
		return decision{Route: routeHeartbeat}
	default:
		return decision{Route: routeCode}
	}
}

// queryFlag reports whether a query parameter is a truthy flag.
//
// It mirrors the reference server's _query_flag helper: the last value wins,
// the value is trimmed and lowercased, and it counts as set only when the
// result is one of "", "1", "true", "yes", or "on". A bare key such as
// "?heartbeat" therefore matches, while "?heartbeat=0" and "?start=false" do
// not. url.Values.Get returns the first value, so the slice is indexed
// directly to honor the documented last-value rule.
func queryFlag(query url.Values, key string) bool {
	values, ok := query[key]
	if !ok || len(values) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(values[len(values)-1])) {
	case "", "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
