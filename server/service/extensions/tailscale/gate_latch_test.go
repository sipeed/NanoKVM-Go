package tailscale

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGateLatchStore(t *testing.T) {
	latchPath := func(t *testing.T) string {
		return filepath.Join(t.TempDir(), "remote-network", "tailscale-gate.json")
	}

	t.Run("missing latch is not paired", func(t *testing.T) {
		store := newGateLatchStore(latchPath(t))
		if store.paired("net-test") {
			t.Fatal("a missing latch must fail closed")
		}
	})

	t.Run("mark persists across instances and is keyed by network", func(t *testing.T) {
		path := latchPath(t)
		store := newGateLatchStore(path)
		if err := store.mark("net-test", time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)); err != nil {
			t.Fatalf("mark: %v", err)
		}

		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("lstat: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("latch mode = %o, want 0600", info.Mode().Perm())
		}

		reloaded := newGateLatchStore(path)
		if !reloaded.paired("net-test") {
			t.Fatal("a fresh instance must see the persisted latch")
		}
		if reloaded.paired("net-other") {
			t.Fatal("the latch must not satisfy a different network")
		}
		if reloaded.paired("") {
			t.Fatal("the latch must not satisfy an empty network")
		}
	})

	t.Run("clear removes the latch and tolerates a missing file", func(t *testing.T) {
		path := latchPath(t)
		store := newGateLatchStore(path)
		if err := store.mark("net-test", time.Now()); err != nil {
			t.Fatalf("mark: %v", err)
		}
		if err := store.clear(); err != nil {
			t.Fatalf("clear: %v", err)
		}
		if store.paired("net-test") {
			t.Fatal("cleared latch must not pair")
		}
		if err := store.clear(); err != nil {
			t.Fatalf("clear of a missing latch: %v", err)
		}
	})

	t.Run("corrupt content fails closed", func(t *testing.T) {
		path := latchPath(t)
		store := newGateLatchStore(path)
		if err := store.mark("net-test", time.Now()); err != nil {
			t.Fatalf("mark: %v", err)
		}
		if err := os.WriteFile(path, []byte(`{"version":1,"networkId":"net-test"`), 0o600); err != nil {
			t.Fatalf("corrupt: %v", err)
		}
		if store.paired("net-test") {
			t.Fatal("a truncated latch must fail closed")
		}
	})

	t.Run("wrong permissions fail closed", func(t *testing.T) {
		path := latchPath(t)
		store := newGateLatchStore(path)
		if err := store.mark("net-test", time.Now()); err != nil {
			t.Fatalf("mark: %v", err)
		}
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		if store.paired("net-test") {
			t.Fatal("a world-readable latch must fail closed")
		}
	})

	t.Run("unknown fields and versions fail closed", func(t *testing.T) {
		path := latchPath(t)
		store := newGateLatchStore(path)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("{\"version\":2,\"networkId\":\"net-test\",\"peerPaired\":true}\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if store.paired("net-test") {
			t.Fatal("an unknown version must fail closed")
		}
		if err := os.WriteFile(path, []byte("{\"version\":1,\"networkId\":\"net-test\",\"peerPaired\":true,\"extra\":1}\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if store.paired("net-test") {
			t.Fatal("an unknown field must fail closed")
		}
	})
}
