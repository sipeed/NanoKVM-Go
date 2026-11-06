package config

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultConfigUsesNormalFingerRobotMode(t *testing.T) {
	if got := defaultConfig.FingerRobot; got != FingerRobotModeNormal {
		t.Fatalf("default fingerRobot mode = %q, want %q", got, FingerRobotModeNormal)
	}
}

func TestFingerRobotModeParsing(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want FingerRobotMode
	}{
		{
			name: "legacy config without field",
			yaml: "proto: http\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "empty value",
			yaml: "fingerRobot:\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "quoted empty value",
			yaml: "fingerRobot: \"\"\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "whitespace only value",
			yaml: "fingerRobot: \"   \"\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "normal",
			yaml: "fingerRobot: normal\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "debug",
			yaml: "fingerRobot: debug\n",
			want: FingerRobotModeDebug,
		},
		{
			name: "invalid value",
			yaml: "fingerRobot: maintenance\n",
			want: FingerRobotModeNormal,
		},
		{
			name: "padded debug is not accepted",
			yaml: "fingerRobot: \" debug \"\n",
			want: FingerRobotModeNormal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseFingerRobotModeForTest(t, test.yaml)
			if got != test.want {
				t.Fatalf("fingerRobot mode = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInvalidFingerRobotModeLogsAndNormalizes(t *testing.T) {
	var output bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")

	got := normalizeFingerRobotMode("maintenance")
	if got != FingerRobotModeNormal {
		t.Fatalf("invalid fingerRobot mode = %q, want %q", got, FingerRobotModeNormal)
	}

	wantLog := `ERROR: invalid fingerRobot mode "maintenance"; using "normal"`
	if !strings.Contains(output.String(), wantLog) {
		t.Fatalf("invalid mode log = %q, want it to contain %q", output.String(), wantLog)
	}
}

func parseFingerRobotModeForTest(t *testing.T, data string) FingerRobotMode {
	t.Helper()

	reader := viper.New()
	reader.SetConfigType("yaml")
	reader.SetDefault("fingerRobot", string(FingerRobotModeNormal))
	if err := reader.ReadConfig(strings.NewReader(data)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	var conf Config
	if err := reader.Unmarshal(&conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	return normalizeFingerRobotMode(conf.FingerRobot)
}
