package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ChronyStatusInfo struct {
	IsSynchronized bool
	LastSyncTime   int64
	LeapStatus     string
	Stratum        string
	ReferenceID    string
}

var runChronycCommand = func(args ...string) ([]byte, error) {
	return exec.Command("chronyc", args...).CombinedOutput()
}

func ChronycImmediateSync() error {
	if err := runChronycCommandExpectOK("makestep", "0.1", "1"); err != nil {
		return err
	}
	return runChronycCommandExpectOK("burst", "1/2")
}

func runChronycCommandExpectOK(args ...string) error {
	output, err := runChronycCommand(args...)
	outputStr := string(output)
	command := "chronyc"
	if len(args) > 0 {
		command += " " + args[0]
	}

	if err != nil {
		return fmt.Errorf("[%s] Execution failed: %w, output: %s", command, err, outputStr)
	}

	if !strings.Contains(outputStr, "200 OK") {
		return fmt.Errorf("[%s] Not the expected result: %s", command, outputStr)
	}

	return nil
}

func GetChronySyncStatus() (*ChronyStatusInfo, error) {
	cmd := exec.Command("chronyc", "tracking")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("[chronyc tracking] Execution failed: %w, output: %s", err, stderr.String())
	}

	return parseChronyTrackingOutput(out.String())
}

func parseChronyTrackingOutput(output string) (*ChronyStatusInfo, error) {
	info := &ChronyStatusInfo{}
	lines := strings.Split(output, "\n")
	parsedFields := 0

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Reference ID":
			info.ReferenceID = value
			parsedFields++
		case "Stratum":
			info.Stratum = value
			parsedFields++
		case "Ref time (UTC)":
			parsedTime, err := time.Parse(time.ANSIC, value)
			if err != nil {
				return nil, fmt.Errorf("[chronyc tracking] Parse Ref time (UTC) failed: %w", err)
			}
			info.LastSyncTime = parsedTime.UnixMilli()
			parsedFields++
		case "Leap status":
			info.LeapStatus = value
			parsedFields++
		}
	}

	if parsedFields < 4 {
		return nil, errors.New("[chronyc tracking] Parse output failed")
	}

	if info.LeapStatus == "Normal" {
		info.IsSynchronized = true
	}

	return info, nil
}
