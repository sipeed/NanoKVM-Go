package storageguard

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLocalAccessExcludesStorageTransitions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	first, err := acquire(path, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := acquire(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if lock, err := acquire(path, true); !errors.Is(err, ErrBusy) {
		if lock != nil {
			lock.Close()
		}
		t.Fatalf("exclusive lock during file access: %v", err)
	}
	first.Close()
	second.Close()
	transition, err := acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer transition.Close()
	if lock, err := acquire(path, false); !errors.Is(err, ErrBusy) {
		if lock != nil {
			lock.Close()
		}
		t.Fatalf("file access during transition: %v", err)
	}
}

func TestChildKeepsInheritedTransactionAfterParentCloses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	lock, err := acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	command := exec.Command("/bin/sh", "-c", "flock -n -x 3 || exit 1; echo locked; read release")
	lock.Attach(command)
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill() })
	if line, err := bufio.NewReader(output).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("child did not reuse lock: %q, %v", line, err)
	}
	lock.Close()
	if other, err := acquire(path, true); !errors.Is(err, ErrBusy) {
		if other != nil {
			other.Close()
		}
		t.Fatalf("child lost lock when parent closed: %v", err)
	}
	_, _ = input.Write([]byte("done\n"))
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	other, err := acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
}

func TestLocalAccessValidatesDataMountAndUSBState(t *testing.T) {
	for _, test := range []struct {
		name, options, backing, mount string
		write                         bool
		want                          error
	}{
		{name: "read local", options: "ro", mount: "exfat", write: false},
		{name: "write local", options: "rw", mount: "exfat", write: true},
		{name: "write read-only", options: "ro", mount: "exfat", write: true, want: ErrReadOnly},
		{name: "USB overrides writable mount", options: "rw", mount: "exfat", backing: "/exfat.img", write: true, want: ErrReadOnly},
		{name: "read during USB sharing", options: "ro", mount: "exfat", backing: "/exfat.img"},
		{name: "unmounted must not fall back to rootfs", write: true, want: ErrUnavailable},
		{name: "wrong filesystem", options: "rw", mount: "ext4", write: true, want: ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			paths := storagePaths{image: filepath.Join(root, "image"), directory: "/data", lun: filepath.Join(root, "lun"), mountInfo: filepath.Join(root, "mountinfo"), devices: filepath.Join(root, "devices")}
			writeTestFile(t, paths.image, "image")
			writeTestFile(t, paths.lun, test.backing)
			mountInfo := ""
			if test.mount != "" {
				mountInfo = "42 1 7:0 / /data " + test.options + " - " + test.mount + " /dev/loop0 " + test.options + "\n"
			}
			writeTestFile(t, paths.mountInfo, mountInfo)
			writeTestFile(t, filepath.Join(paths.devices, "7:0/loop/backing_file"), paths.image+"\n")
			if err := checkLocal(paths, test.write); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func localRecoveryFixture(t *testing.T) (storagePaths, string) {
	t.Helper()
	root := t.TempDir()
	paths := storagePaths{
		image: filepath.Join(root, "image"), directory: "/data",
		lun: filepath.Join(root, "disk1"), imageLun: filepath.Join(root, "disk0"),
		usbMarker: filepath.Join(root, "usb-owner"),
		mountInfo: filepath.Join(root, "mountinfo"), devices: filepath.Join(root, "devices"),
	}
	writeTestFile(t, paths.image, "image")
	writeTestFile(t, paths.lun, "\n")
	writeTestFile(t, paths.imageLun, "\n")
	writeTestFile(t, paths.usbMarker, "")
	writeTestFile(t, paths.mountInfo, "42 1 7:0 / /data ro - exfat /dev/loop0 ro\n")
	writeTestFile(t, filepath.Join(paths.devices, "7:0/loop/backing_file"), paths.image+"\n")
	return paths, filepath.Join(root, "lock")
}

func TestHostEjectRecoversLocalWritesAndRetainsSharedLease(t *testing.T) {
	paths, lockPath := localRecoveryFixture(t)
	restored := 0
	release, err := local(paths, lockPath, true, func(lock *Lock) error {
		restored++
		if other, err := acquire(lockPath, false); !errors.Is(err, ErrBusy) {
			if other != nil {
				other.Close()
			}
			t.Fatalf("recovery did not exclude file access: %v", err)
		}
		writeTestFile(t, paths.mountInfo, "42 1 7:0 / /data rw - exfat /dev/loop0 rw\n")
		return os.Remove(paths.usbMarker)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if restored != 1 {
		t.Fatalf("recovery calls = %d", restored)
	}
	reader, err := acquire(lockPath, false)
	if err != nil {
		t.Fatalf("recovered operation blocked another local reader: %v", err)
	}
	reader.Close()
	if transition, err := acquire(lockPath, true); !errors.Is(err, ErrBusy) {
		if transition != nil {
			transition.Close()
		}
		t.Fatalf("write lost its storage lease: %v", err)
	}
	// Subsequent chunks/writes use the normal shared lease without remounting.
	next, err := local(paths, lockPath, true, func(*Lock) error {
		t.Fatal("already writable storage was remounted")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	next()
}

func TestAutomaticRecoveryPreservesStorageOwnership(t *testing.T) {
	for _, test := range []struct {
		name   string
		write  bool
		change func(*testing.T, storagePaths)
		want   error
	}{
		{name: "reads keep the current mount"},
		{name: "USB still attached", write: true, want: ErrReadOnly, change: func(t *testing.T, paths storagePaths) {
			writeTestFile(t, paths.lun, paths.image)
		}},
		{name: "image still attached", write: true, want: ErrReadOnly, change: func(t *testing.T, paths storagePaths) {
			writeTestFile(t, paths.imageLun, "/data/rescue.img")
		}},
		{name: "read-only without a USB handoff", write: true, want: ErrReadOnly, change: func(t *testing.T, paths storagePaths) {
			if err := os.Remove(paths.usbMarker); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "unmounted storage", write: true, want: ErrUnavailable, change: func(t *testing.T, paths storagePaths) {
			writeTestFile(t, paths.mountInfo, "")
		}},
		{name: "unexpected filesystem", write: true, want: ErrUnavailable, change: func(t *testing.T, paths storagePaths) {
			writeTestFile(t, paths.mountInfo, "42 1 7:0 / /data ro - ext4 /dev/loop0 ro\n")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			paths, lockPath := localRecoveryFixture(t)
			if test.change != nil {
				test.change(t, paths)
			}
			release, err := local(paths, lockPath, test.write, func(*Lock) error {
				t.Fatal("unsafe or unnecessary recovery")
				return nil
			})
			if release != nil {
				release()
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("access error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestAutomaticRecoveryWaitsForActiveReaders(t *testing.T) {
	paths, lockPath := localRecoveryFixture(t)
	reader, err := acquire(lockPath, false)
	if err != nil {
		t.Fatal(err)
	}
	restore := func(*Lock) error {
		writeTestFile(t, paths.mountInfo, "42 1 7:0 / /data rw - exfat /dev/loop0 rw\n")
		return nil
	}
	if release, err := local(paths, lockPath, true, restore); !errors.Is(err, ErrBusy) {
		if release != nil {
			release()
		}
		t.Fatalf("recovered while another reader held the mount: %v", err)
	}
	reader.Close()
	release, err := local(paths, lockPath, true, restore)
	if err != nil {
		t.Fatalf("retry after reader finished: %v", err)
	}
	release()
}

func TestRecoveryRechecksOwnershipAfterExchangingLocks(t *testing.T) {
	for _, disk := range []string{"disk0", "disk1"} {
		t.Run(disk, func(t *testing.T) {
			paths, lockPath := localRecoveryFixture(t)
			ready, err := canRestoreLocal(paths)
			if err != nil || !ready {
				t.Fatalf("initial handoff: ready=%v, err=%v", ready, err)
			}
			path := paths.lun
			if disk == "disk0" {
				path = paths.imageLun
			}
			// Another USB transaction attached media before recovery got its lock.
			writeTestFile(t, path, paths.image)
			lock, err := acquire(lockPath, true)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			err = restoreLocalAccess(paths, lock, func(*Lock) error {
				t.Fatal("recovery ignored a newly attached medium")
				return nil
			})
			if !errors.Is(err, ErrReadOnly) {
				t.Fatalf("recovery after reattach: %v", err)
			}
		})
	}
}

func TestAutomaticRecoveryFailureDoesNotGrantWriteAccess(t *testing.T) {
	failed := errors.New("mount failed")
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "command failed", err: failed, want: failed},
		{name: "mount stayed read-only", want: ErrReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			paths, lockPath := localRecoveryFixture(t)
			release, err := local(paths, lockPath, true, func(*Lock) error { return test.err })
			if release != nil || !errors.Is(err, test.want) {
				t.Fatalf("access after failed recovery: lease=%v, err=%v", release != nil, err)
			}
			lock, err := acquire(lockPath, true)
			if err != nil {
				t.Fatalf("failed recovery leaked its lock: %v", err)
			}
			lock.Close()
		})
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
