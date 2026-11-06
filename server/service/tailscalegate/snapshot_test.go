package tailscalegate

import (
	"strings"
	"testing"
)

func stringPtr(value string) *string { return &value }

func boolPtr(value bool) *bool { return &value }

func TestResolveDefaultsForEmptyReport(t *testing.T) {
	got := Resolve(Report{})
	want := Snapshot{
		Status: StatusOffline,
		Start:  false,
		Match:  false,
		Name:   "",
		IP:     "",
	}
	if got != want {
		t.Fatalf("Resolve(Report{}) = %+v, want %+v", got, want)
	}
}

func TestResolveStatusValues(t *testing.T) {
	cases := []struct {
		name  string
		input *string
		want  Status
	}{
		{name: "exact online", input: stringPtr("online"), want: StatusOnline},
		{name: "trimmed online", input: stringPtr(" online "), want: StatusOnline},
		{name: "uppercase is not online", input: stringPtr("ONLINE"), want: StatusOffline},
		{name: "offline", input: stringPtr("offline"), want: StatusOffline},
		{name: "other value", input: stringPtr("starting"), want: StatusOffline},
		{name: "empty string", input: stringPtr(""), want: StatusOffline},
		{name: "nil", input: nil, want: StatusOffline},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(Report{Status: tc.input})
			if got.Status != tc.want {
				t.Fatalf("Resolve status = %q, want %q", got.Status, tc.want)
			}
		})
	}
}

// TestResolveFieldIndependence proves that a single missing field only
// affects that field: the others keep the values the driver supplied.
func TestResolveFieldIndependence(t *testing.T) {
	full := Report{
		Status: stringPtr("online"),
		Start:  boolPtr(true),
		Match:  boolPtr(true),
		Name:   stringPtr("peer"),
		IP:     stringPtr("100.64.0.1"),
	}

	cleared := func(mutate func(*Report)) Report {
		report := full
		mutate(&report)
		return report
	}

	cases := []struct {
		name string
		in   Report
		want Snapshot
	}{
		{
			name: "only status missing",
			in:   cleared(func(r *Report) { r.Status = nil }),
			want: Snapshot{Status: StatusOffline, Start: true, Match: true},
		},
		{
			name: "only start missing",
			in:   cleared(func(r *Report) { r.Start = nil }),
			want: Snapshot{Status: StatusOnline, Start: false, Match: true, Name: "peer", IP: "100.64.0.1"},
		},
		{
			name: "only match missing",
			in:   cleared(func(r *Report) { r.Match = nil }),
			want: Snapshot{Status: StatusOnline, Start: true, Match: false, Name: "peer", IP: "100.64.0.1"},
		},
		{
			name: "only name missing",
			in:   cleared(func(r *Report) { r.Name = nil }),
			want: Snapshot{Status: StatusOnline, Start: true, Match: true, Name: "", IP: "100.64.0.1"},
		},
		{
			name: "only ip missing",
			in:   cleared(func(r *Report) { r.IP = nil }),
			want: Snapshot{Status: StatusOnline, Start: true, Match: true, Name: "peer", IP: ""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.in); got != tc.want {
				t.Fatalf("Resolve = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestResolveNameIPConvergence(t *testing.T) {
	cases := []struct {
		name   string
		report Report
		want   Snapshot
	}{
		{
			name: "online keeps both",
			report: Report{
				Status: stringPtr("online"),
				Name:   stringPtr("peer"),
				IP:     stringPtr("100.64.0.1"),
			},
			want: Snapshot{Status: StatusOnline, Name: "peer", IP: "100.64.0.1"},
		},
		{
			name: "offline drops both",
			report: Report{
				Status: stringPtr("offline"),
				Name:   stringPtr("peer"),
				IP:     stringPtr("100.64.0.1"),
			},
			want: Snapshot{Status: StatusOffline},
		},
		{
			name: "missing status drops both",
			report: Report{
				Name: stringPtr("peer"),
				IP:   stringPtr("100.64.0.1"),
			},
			want: Snapshot{Status: StatusOffline},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.report); got != tc.want {
				t.Fatalf("Resolve = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestResolveFiltersAndTruncatesNameIP(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "drops nul and delete", input: "\x00Ab\x7f", want: "Ab"},
		{name: "keeps spaces and tilde", input: " a~b ", want: " a~b "},
		{name: "empty stays empty", input: "", want: ""},
		{
			name:  "multi byte utf8 is dropped bytewise",
			input: "\u00e9\x00",
			want:  "",
		},
		{
			name:  "exactly 64 bytes passes through",
			input: strings.Repeat("a", 64),
			want:  strings.Repeat("a", 64),
		},
		{name: "truncates to 64 bytes", input: strings.Repeat("a", 100), want: strings.Repeat("a", 64)},
		{
			name:  "filters before truncating",
			input: strings.Repeat("a", 70) + strings.Repeat("\x00", 30),
			want:  strings.Repeat("a", 64),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := Report{
				Status: stringPtr("online"),
				Name:   stringPtr(tc.input),
				IP:     stringPtr(tc.input),
			}
			got := Resolve(report)
			if got.Name != tc.want {
				t.Fatalf("Resolve name = %q, want %q", got.Name, tc.want)
			}
			if got.IP != tc.want {
				t.Fatalf("Resolve ip = %q, want %q", got.IP, tc.want)
			}
		})
	}
}

func TestSnapshotGateProjection(t *testing.T) {
	snap := Snapshot{
		Status: StatusOnline,
		Start:  true,
		Match:  false,
		Name:   "peer",
		IP:     "100.64.0.1",
	}
	want := GateState{Start: true, Match: false}
	if got := snap.Gate(); got != want {
		t.Fatalf("Gate() = %+v, want %+v", got, want)
	}
}

func TestGateStateConfirmReplacesWholeValue(t *testing.T) {
	initial := GateState{Start: true, Match: true}
	next := Snapshot{Status: StatusOnline, Start: false, Match: true}

	got := initial.Confirm(next)
	want := GateState{Start: false, Match: true}
	if got != want {
		t.Fatalf("Confirm() = %+v, want %+v", got, want)
	}
	// A value receiver must not mutate the state the caller still holds, which
	// is what lets a failed refresh simply skip Confirm and keep the old gate.
	if initial != (GateState{Start: true, Match: true}) {
		t.Fatalf("Confirm mutated receiver to %+v", initial)
	}
}

func TestGateStateConfirmAcceptsZeroSnapshot(t *testing.T) {
	initial := GateState{Start: true, Match: true}
	if got := initial.Confirm(Snapshot{Status: StatusOffline}); got != (GateState{}) {
		t.Fatalf("Confirm(zero snapshot) = %+v, want zero gate state", got)
	}
}
