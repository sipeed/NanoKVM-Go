package ai

import (
	"bytes"
	"testing"
)

func TestServiceStateRuntimeMCPEnablesSnapshotOnly(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)

	manager, _ := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	if _, err := manager.Set(aiServiceMCP, true, "web"); err != nil {
		t.Fatalf("set mcp enabled: %v", err)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:false:false:true:0s"}) {
		t.Fatalf("expected snapshot-only runtime, got %#v", runtimeCalls)
	}
}

func TestServiceStateRuntimeKeepsSnapshotWhenMCPDisabledButScreenTimelapseEnabled(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)

	manager, path := newTestServiceStateManager(t)
	initial := defaultServiceStateSnapshot()
	initial.Revision = 7
	initial.Services[aiServiceMCP] = ServiceStateEntry{Enabled: true, UpdatedAt: "2026-06-18T00:00:00Z", Source: "test"}
	initial.Services[aiServiceScreenTimelapse] = ServiceStateEntry{Enabled: true, UpdatedAt: "2026-06-18T00:00:00Z", Source: "test"}
	if err := saveServiceStateToPath(path, initial); err != nil {
		t.Fatalf("write initial state: %v", err)
	}
	if err := manager.Load(); err != nil {
		t.Fatalf("load initial state: %v", err)
	}
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	if _, err := manager.Set(aiServiceMCP, false, "web"); err != nil {
		t.Fatalf("set mcp disabled: %v", err)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected snapshot to remain enabled while timelapse stays enabled, got %#v", runtimeCalls)
	}
}

func TestServiceStateRuntimeScreenTimelapseEnablesSnapshotAndTimelapse(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)

	manager, _ := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	if _, err := manager.Set(aiServiceScreenTimelapse, true, "web"); err != nil {
		t.Fatalf("set screen timelapse enabled: %v", err)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected snapshot and timelapse runtime, got %#v", runtimeCalls)
	}
}

func TestServiceStateRuntimeDisablesTimelapseButKeepsSnapshotWhenMCPEnabled(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)

	manager, path := newTestServiceStateManager(t)
	initial := defaultServiceStateSnapshot()
	initial.Revision = 7
	initial.Services[aiServiceMCP] = ServiceStateEntry{Enabled: true, UpdatedAt: "2026-06-18T00:00:00Z", Source: "test"}
	initial.Services[aiServiceScreenTimelapse] = ServiceStateEntry{Enabled: true, UpdatedAt: "2026-06-18T00:00:00Z", Source: "test"}
	if err := saveServiceStateToPath(path, initial); err != nil {
		t.Fatalf("write initial state: %v", err)
	}
	if err := manager.Load(); err != nil {
		t.Fatalf("load initial state: %v", err)
	}
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	if _, err := manager.Set(aiServiceScreenTimelapse, false, "web"); err != nil {
		t.Fatalf("set screen timelapse disabled: %v", err)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:false:false:true:0s"}) {
		t.Fatalf("expected timelapse disabled while snapshot remains enabled, got %#v", runtimeCalls)
	}
}

func TestServiceStateRuntimeFailureDoesNotPersistOrBroadcast(t *testing.T) {
	var runtimeCalls []string
	setTestAIRuntime(t, errTestAIRuntime, &runtimeCalls)

	manager, path := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	if err := manager.Save(); err != nil {
		t.Fatalf("save initial state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, path)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceMCP, true, "web"); err == nil {
		t.Fatalf("expected AI runtime error")
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:false:false:true:0s"}) {
		t.Fatalf("expected AI runtime call, got %#v", runtimeCalls)
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after runtime failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMCP] != beforeState.Services[aiServiceMCP] {
		t.Fatalf("expected state unchanged, before=%#v after=%#v", beforeState, afterState)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after runtime failure: %#v", event)
	default:
	}
}
