package config

import "testing"

func TestParseDeviceNumber(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "raw payload",
			output: "  NAG-DEVICE-001\n",
			want:   "NAG-DEVICE-001",
		},
		{
			name:   "legacy A label",
			output: "[A] NAG-DEVICE-A",
			want:   "NAG-DEVICE-A",
		},
		{
			name:   "legacy B label",
			output: "[B] NAG-DEVICE-B",
			want:   "NAG-DEVICE-B",
		},
		{
			name:   "legacy C label",
			output: "[C] NAG-DEVICE-C",
			want:   "NAG-DEVICE-C",
		},
		{
			name:   "other legacy letter label",
			output: "[Z] NAG-DEVICE-Z",
			want:   "NAG-DEVICE-Z",
		},
		{
			name:   "NanoKVM Go Plus model and region",
			output: "NanoKVM-Go-Plus[A]NibD10012",
			want:   "NibD10012",
		},
		{
			name:   "NanoKVM Go model and region",
			output: "NanoKVM-Go[B] NibD10013",
			want:   "NibD10013",
		},
		{
			name:   "empty output",
			output: " \n\t ",
			want:   unknownDeviceNumber,
		},
		{
			name:   "legacy label without payload",
			output: "[C] ",
			want:   unknownDeviceNumber,
		},
		{
			name:   "model and label without payload",
			output: "NanoKVM-Go-Plus[A] ",
			want:   unknownDeviceNumber,
		},
		{
			name:   "brackets in raw payload are preserved",
			output: "NAG[ABC]DEVICE",
			want:   "NAG[ABC]DEVICE",
		},
		{
			name:   "other model prefix is preserved",
			output: "OtherModel[A]NibD10014",
			want:   "OtherModel[A]NibD10014",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDeviceNumber(tt.output); got != tt.want {
				t.Errorf("parseDeviceNumber(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
