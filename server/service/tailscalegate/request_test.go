package tailscalegate

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDispatchMethodAndPath(t *testing.T) {
	const trailingSlash = LoginCodePath + "/"

	cases := []struct {
		name       string
		method     string
		target     string
		wantRoute  route
		wantStatus int
	}{
		{
			name:       "get exact path with no query is code",
			method:     http.MethodGet,
			target:     LoginCodePath,
			wantRoute:  routeCode,
			wantStatus: 0,
		},
		{
			name:       "post login path is 400 and checked before the path",
			method:     http.MethodPost,
			target:     LoginCodePath,
			wantRoute:  routeUnknown,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "put login path is 400",
			method:     http.MethodPut,
			target:     LoginCodePath,
			wantRoute:  routeUnknown,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete login path is 400",
			method:     http.MethodDelete,
			target:     LoginCodePath,
			wantRoute:  routeUnknown,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "post unknown path is still 400",
			method:     http.MethodPost,
			target:     "/api/unknown",
			wantRoute:  routeUnknown,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get unknown path is 404",
			method:     http.MethodGet,
			target:     "/api/unknown",
			wantRoute:  routeUnknown,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get trailing slash variant is 404",
			method:     http.MethodGet,
			target:     trailingSlash,
			wantRoute:  routeUnknown,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get login path with a query still routes",
			method:     http.MethodGet,
			target:     LoginCodePath + "?heartbeat=1",
			wantRoute:  routeHeartbeat,
			wantStatus: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.target, nil)
			got := dispatch(request)
			if got.Route != tc.wantRoute || got.Status != tc.wantStatus {
				t.Fatalf("dispatch(%s %s) = %+v, want route=%v status=%d",
					tc.method, tc.target, got, tc.wantRoute, tc.wantStatus)
			}
		})
	}
}

func TestDispatchNilRequestIsBadRequest(t *testing.T) {
	got := dispatch(nil)
	if got.Route != routeUnknown || got.Status != http.StatusBadRequest {
		t.Fatalf("dispatch(nil) = %+v, want routeUnknown status=%d", got, http.StatusBadRequest)
	}
}

// A request built without a URL cannot match LoginCodePath, so it is rejected
// as an unknown path rather than panicking on the nil dereference.
func TestDispatchNilURLIsNotFound(t *testing.T) {
	got := dispatch(&http.Request{Method: http.MethodGet})
	if got.Route != routeUnknown || got.Status != http.StatusNotFound {
		t.Fatalf("dispatch(nil URL) = %+v, want routeUnknown status=%d", got, http.StatusNotFound)
	}
}

func TestDispatchQueryPrecedence(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantRoute route
	}{
		{name: "no query is code", query: "", wantRoute: routeCode},
		{name: "start wins over heartbeat", query: "?heartbeat=true&start=true", wantRoute: routeStart},
		{name: "exit wins over disable", query: "?exit=true&disable=true", wantRoute: routeExit},
		{name: "exit wins over heartbeat", query: "?exit=true&heartbeat=true", wantRoute: routeExit},
		{name: "disable wins over heartbeat", query: "?disable=true&heartbeat=true", wantRoute: routeDisable},
		{name: "start wins over everything", query: "?disable=true&heartbeat=true&exit=true&start=true", wantRoute: routeStart},
		{name: "exit alone", query: "?exit=true", wantRoute: routeExit},
		{name: "legacy on is no longer routed", query: "?on=true", wantRoute: routeCode},
		{name: "legacy off is no longer routed", query: "?off=true", wantRoute: routeCode},
		{name: "heartbeat alone", query: "?heartbeat=true", wantRoute: routeHeartbeat},
		{name: "heartbeat accepts 1", query: "?heartbeat=1", wantRoute: routeHeartbeat},
		{name: "bare heartbeat key matches", query: "?heartbeat", wantRoute: routeHeartbeat},
		{name: "blank heartbeat value matches", query: "?heartbeat=", wantRoute: routeHeartbeat},
		{name: "whitespace heartbeat value matches", query: "?heartbeat=%20", wantRoute: routeHeartbeat},
		{name: "padded heartbeat value matches", query: "?heartbeat=+TRUE+", wantRoute: routeHeartbeat},
		{name: "uppercase YES matches", query: "?heartbeat=YES", wantRoute: routeHeartbeat},
		{name: "ON matches", query: "?heartbeat=on", wantRoute: routeHeartbeat},
		{name: "start=false does not match and falls through to code", query: "?start=false", wantRoute: routeCode},
		{name: "exit=false does not match and falls through to code", query: "?exit=false", wantRoute: routeCode},
		{name: "heartbeat=0 does not match", query: "?heartbeat=0", wantRoute: routeCode},
		{name: "heartbeat=no does not match", query: "?heartbeat=no", wantRoute: routeCode},
		{name: "heartbeat=off does not match", query: "?heartbeat=off", wantRoute: routeCode},
		{name: "last heartbeat value wins", query: "?heartbeat=0&heartbeat=1", wantRoute: routeHeartbeat},
		{name: "last heartbeat value can also lose", query: "?heartbeat=1&heartbeat=0", wantRoute: routeCode},
		{name: "start=false with heartbeat=1 still uses heartbeat", query: "?start=false&heartbeat=1", wantRoute: routeHeartbeat},
		{name: "consent alone is not a route and falls through to code", query: "?consent=true", wantRoute: routeCode},
		{name: "start with consent is still start", query: "?start=true&consent=true", wantRoute: routeStart},
		{name: "exit with consent keeps exit", query: "?exit=true&consent=true", wantRoute: routeExit},
		{name: "disable with consent keeps disable", query: "?disable=true&consent=true", wantRoute: routeDisable},
		{name: "heartbeat with consent keeps heartbeat", query: "?heartbeat=true&consent=true", wantRoute: routeHeartbeat},
		{name: "unknown query is code", query: "?other=true", wantRoute: routeCode},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, LoginCodePath+tc.query, nil)
			got := dispatch(request)
			if got.Route != tc.wantRoute || got.Status != 0 {
				t.Fatalf("dispatch(%q) = %+v, want route=%v status=0", tc.query, got, tc.wantRoute)
			}
		})
	}
}

// TestActionMessagesFitTheWireBudget pins the two messages new to the consent
// channel: they are the exact documented vocabulary, printable ASCII, within
// the ten-byte ceiling, and unchanged by the boundedMessage filter that every
// driver-provided message passes through.
func TestActionMessagesFitTheWireBudget(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{name: "consent", message: MessageConsent, want: "CONSENT"},
		{name: "disabled", message: MessageDisabled, want: "DISABLED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.message != tc.want {
				t.Fatalf("message = %q, want %q", tc.message, tc.want)
			}
			if len(tc.message) > maxMessageBytes {
				t.Fatalf("len(%q) = %d, want <= %d", tc.message, len(tc.message), maxMessageBytes)
			}
			for _, r := range tc.message {
				if r < 0x20 || r > 0x7e {
					t.Fatalf("message %q contains non printable ASCII %q", tc.message, r)
				}
			}
			if got := boundedMessage(tc.message); got != tc.message {
				t.Fatalf("boundedMessage(%q) = %q, want unchanged", tc.message, got)
			}
		})
	}
}

func TestQueryFlag(t *testing.T) {
	cases := []struct {
		name  string
		query url.Values
		key   string
		want  bool
	}{
		{name: "absent key", query: url.Values{}, key: "heartbeat", want: false},
		{name: "empty value list", query: url.Values{"heartbeat": {}}, key: "heartbeat", want: false},
		{name: "bare empty string", query: url.Values{"heartbeat": {""}}, key: "heartbeat", want: true},
		{name: "one", query: url.Values{"heartbeat": {"1"}}, key: "heartbeat", want: true},
		{name: "true", query: url.Values{"heartbeat": {"true"}}, key: "heartbeat", want: true},
		{name: "yes", query: url.Values{"heartbeat": {"yes"}}, key: "heartbeat", want: true},
		{name: "on", query: url.Values{"heartbeat": {"on"}}, key: "heartbeat", want: true},
		{name: "uppercase", query: url.Values{"heartbeat": {"TRUE"}}, key: "heartbeat", want: true},
		{name: "padded", query: url.Values{"heartbeat": {"  On  "}}, key: "heartbeat", want: true},
		{name: "tab padded", query: url.Values{"heartbeat": {"\t1\n"}}, key: "heartbeat", want: true},
		{name: "zero", query: url.Values{"heartbeat": {"0"}}, key: "heartbeat", want: false},
		{name: "false", query: url.Values{"heartbeat": {"false"}}, key: "heartbeat", want: false},
		{name: "no", query: url.Values{"heartbeat": {"no"}}, key: "heartbeat", want: false},
		{name: "off", query: url.Values{"heartbeat": {"off"}}, key: "heartbeat", want: false},
		{name: "numeric other", query: url.Values{"heartbeat": {"2"}}, key: "heartbeat", want: false},
		{name: "substring", query: url.Values{"heartbeat": {"yesterday"}}, key: "heartbeat", want: false},
		{name: "last value wins true", query: url.Values{"heartbeat": {"0", "1"}}, key: "heartbeat", want: true},
		{name: "last value wins false", query: url.Values{"heartbeat": {"1", "0"}}, key: "heartbeat", want: false},
		{name: "different key", query: url.Values{"start": {"1"}}, key: "heartbeat", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := queryFlag(tc.query, tc.key); got != tc.want {
				t.Fatalf("queryFlag(%v, %q) = %v, want %v", tc.query, tc.key, got, tc.want)
			}
		})
	}
}
