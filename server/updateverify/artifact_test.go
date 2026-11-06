package updateverify

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOpenArtifactSnapshotMapsExactStableBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.bin")
	content := []byte("stable artifact bytes")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := openArtifactSnapshot(path, "test artifact", int64(len(content)), 1024)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snapshot.Close(); err != nil {
			t.Errorf("close snapshot: %v", err)
		}
	}()
	if got := string(snapshot.Bytes()); got != string(content) {
		t.Fatalf("snapshot bytes = %q, want %q", got, content)
	}
	if snapshot.File() == nil {
		t.Fatal("snapshot file is nil")
	}
	if err := snapshot.VerifyStable(); err != nil {
		t.Fatalf("verify stable snapshot: %v", err)
	}
}

func TestOpenArtifactSnapshotRejectsUnsafeFilesAndSizes(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "symlink")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		path         string
		expectedSize int64
		maxSize      int64
		kind         ErrorKind
	}{
		{name: "missing", path: filepath.Join(dir, "missing"), maxSize: 1024, kind: ErrorIO},
		{name: "directory", path: dir, maxSize: 1024, kind: ErrorPolicy},
		{name: "symlink", path: symlink, maxSize: 1024, kind: ErrorPolicy},
		{name: "fifo", path: fifo, maxSize: 1024, kind: ErrorPolicy},
		{name: "empty", path: empty, maxSize: 1024, kind: ErrorPolicy},
		{name: "expected size", path: regular, expectedSize: 5, maxSize: 1024, kind: ErrorPolicy},
		{name: "maximum size", path: regular, maxSize: 3, kind: ErrorPolicy},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := openArtifactSnapshot(test.path, "test artifact", test.expectedSize, test.maxSize)
			if snapshot != nil {
				_ = snapshot.Close()
				t.Fatal("snapshot unexpectedly returned")
			}
			assertKind(t, err, test.kind)
		})
	}
}

// artifactSnapshot.VerifyStable 的判定依据只有 stat identity
// （device/inode/links/mode/size/mtime/ctime），不含文件内容。因此只有会改变
// 上述字段的变异才可能被确定性检出；本用例覆盖尺寸变化与路径替换两个通道。
// 边界认定见 .trellis/spec/backend/device-and-system-boundaries.md 的
// "Scenario: updateverify 构件稳定性判定边界" 小节。
func TestArtifactSnapshotDetectsSizeMutationAndPathReplacement(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, path string)
	}{
		{
			name: "in-place append",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("X")); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "truncation",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Truncate(path, 4); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "path replacement",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				replacement := path + ".replacement"
				if err := os.WriteFile(replacement, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "artifact")
			if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			// Back-date the artifact so a same-size in-place mutation is
			// guaranteed to move mtime even when the coarse-grained file
			// timestamp clock has not ticked since the file was created.
			past := time.Now().Add(-time.Hour)
			if err := os.Chtimes(path, past, past); err != nil {
				t.Fatal(err)
			}
			snapshot, err := openArtifactSnapshot(path, "test artifact", 8, 1024)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = snapshot.Close() }()
			test.mutate(t, path)
			assertKind(t, snapshot.VerifyStable(), ErrorPolicy)
		})
	}
}

// VerifyStable 的判据只有 stat identity（device/inode/links/mode/size/mtime/ctime），
// 不含内容；因此“是否检出”完全由 identity 是否变化决定：同一 tick 内、同 inode、
// 同尺寸、元数据不变的原地改写不可检出 —— 这是已接受的检测边界。
//
// 为什么不断言“必然不检出”：Linux 内容写入必然更新 mtime/ctime，是否被观察到取决于写入是否
// 跨过 coarse 时钟的一个 tick，而 ctime 不可由用户态设置，无法构造确定性的“元数据全不变”状态；
// 直接断言 nil 会带来时序 flake（本机 ext4 实测 ~3%/次；-count=20 下约 46% 变红。该概率随内核
// tick 粒度与主机速度变化，不是跨平台常数）。一致性律与 tick 无关。
//
// 若将来把判定升级为内容级复检，内容已变而 identity 未变的情形会让本律失效 → 报警提醒，
// 届时应同步更新本注释与 spec 边界章节。边界见
// .trellis/spec/backend/device-and-system-boundaries.md 的
// "Scenario: updateverify 构件稳定性判定边界"。
func TestArtifactSnapshotStabilityVerdictIsIdentityOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := openArtifactSnapshot(path, "test artifact", 8, 1024)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()

	// 真实同尺寸原地改写：只覆盖第 0 字节，尺寸保持 8。
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("X"), 0); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	// 防空转：确认内容确实被改写（若变异退化成 no-op，这里必须失败）。
	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(rewritten) != "Xriginal" {
		t.Fatalf("same-size rewrite did not change content: %q", rewritten)
	}

	var current unix.Stat_t
	if err := unix.Fstat(int(snapshot.File().Fd()), &current); err != nil {
		t.Fatal(err)
	}
	identityChanged := identityFromStat(current) != snapshot.identity
	detected := snapshot.VerifyStable() != nil
	if detected != identityChanged {
		t.Fatalf("stability verdict is not identity-only: detected=%v identityChanged=%v; "+
			"if content-level verification was added, update this test and the documented boundary",
			detected, identityChanged)
	}
}
