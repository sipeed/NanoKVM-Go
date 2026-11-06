package config

import (
	"log"
	"strings"
)

func normalizeFingerRobotMode(mode FingerRobotMode) FingerRobotMode {
	switch mode {
	case "", FingerRobotModeNormal:
		return FingerRobotModeNormal
	case FingerRobotModeDebug:
		return FingerRobotModeDebug
	default:
		if strings.TrimSpace(string(mode)) == "" {
			return FingerRobotModeNormal
		}

		log.Printf(
			"ERROR: invalid fingerRobot mode %q; using %q",
			mode,
			FingerRobotModeNormal,
		)
		return FingerRobotModeNormal
	}
}
