package ai

import (
	"bytes"
	"testing"
)

func TestServiceStateManagerScreenTimelapseRuntimeFailureDoesNotPersistOrBroadcast(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, errTestAIRuntime, &runtimeCalls)

	manager, path := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	if err := manager.Save(); err != nil {
		t.Fatalf("save initial state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, path)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceScreenTimelapse, true, "web"); err == nil {
		t.Fatalf("expected screen timelapse runtime error")
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected AI runtime call, got %#v", runtimeCalls)
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after hook failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceScreenTimelapse] != beforeState.Services[aiServiceScreenTimelapse] {
		t.Fatalf("expected state unchanged, before=%#v after=%#v", beforeState, afterState)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after hook failure: %#v", event)
	default:
	}
}
