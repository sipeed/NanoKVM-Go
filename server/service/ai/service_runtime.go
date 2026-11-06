package ai

import (
	"time"

	"NanoKVM-Server/common"

	log "github.com/sirupsen/logrus"
)

var setKvmCommAIRuntime = common.SetKvmCommAIRuntime

func defaultServiceStateRuntimeReconciler(current ServiceStateSnapshot, next ServiceStateSnapshot, serviceID string, source string) error {
	if serviceID == aiServiceMemoryFabric {
		return reconcileMemoryFabricRuntime(current, next, source)
	}

	return applyServiceStateAIRuntime(next)
}

func reconcileMemoryFabricRuntime(current ServiceStateSnapshot, next ServiceStateSnapshot, source string) error {
	nextMemoryFabricEnabled := next.Services[aiServiceMemoryFabric].Enabled
	currentMemoryFabricEnabled := current.Services[aiServiceMemoryFabric].Enabled

	if nextMemoryFabricEnabled {
		if err := applyServiceStateAIRuntime(next); err != nil {
			return err
		}

		if err := memoryFabricServiceStateHook(true, source); err != nil {
			rollbackServiceStateAIRuntime(current)
			return err
		}

		return nil
	}

	if err := memoryFabricServiceStateHook(false, source); err != nil {
		return err
	}

	if err := applyServiceStateAIRuntime(next); err != nil {
		rollbackMemoryFabricService(currentMemoryFabricEnabled, source)
		return err
	}

	return nil
}

func applyServiceStateAIRuntime(state ServiceStateSnapshot) error {
	snapshotEnabled, snapshotPeriodicEnabled, timelapseEnabled := serviceStateAIRuntimeTargets(state)
	return setKvmCommAIRuntime(snapshotEnabled, snapshotPeriodicEnabled, timelapseEnabled, true, 0*time.Second)
}

func rollbackServiceStateAIRuntime(state ServiceStateSnapshot) {
	if err := applyServiceStateAIRuntime(state); err != nil {
		log.Warnf("failed to rollback AI runtime config: %v", err)
	}
}

func rollbackMemoryFabricService(enabled bool, source string) {
	if err := memoryFabricServiceStateHook(enabled, source); err != nil {
		log.Warnf("failed to rollback Memory Fabric enabled=%t: %v", enabled, err)
	}
}

func serviceStateAIRuntimeTargets(state ServiceStateSnapshot) (snapshotEnabled bool, snapshotPeriodicEnabled bool, timelapseEnabled bool) {
	snapshotEnabled = state.Services[aiServiceMCP].Enabled ||
		state.Services[aiServiceMemoryFabric].Enabled ||
		state.Services[aiServiceScreenTimelapse].Enabled
	snapshotPeriodicEnabled = state.Services[aiServiceMemoryFabric].Enabled ||
		state.Services[aiServiceScreenTimelapse].Enabled
	timelapseEnabled = state.Services[aiServiceScreenTimelapse].Enabled
	return snapshotEnabled, snapshotPeriodicEnabled, timelapseEnabled
}
