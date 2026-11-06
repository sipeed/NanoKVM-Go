package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestBuildCaptiveDNSResponseAnswersOnlyINetA(t *testing.T) {
	for _, test := range []struct {
		name         string
		questionType dnsmessage.Type
		wantAnswers  int
	}{
		{name: "A", questionType: dnsmessage.TypeA, wantAnswers: 1},
		{name: "AAAA is NODATA", questionType: dnsmessage.TypeAAAA, wantAnswers: 0},
		{name: "other IN type is NODATA", questionType: dnsmessage.TypeTXT, wantAnswers: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := captiveDNSQuery(t, dnsmessage.Header{ID: 0x1234, RecursionDesired: true}, []dnsmessage.Question{{
				Name:  dnsmessage.MustNewName("connectivity.example."),
				Type:  test.questionType,
				Class: dnsmessage.ClassINET,
			}})

			responseWire, ok := buildCaptiveDNSResponse(query)
			if !ok {
				t.Fatal("valid query was dropped")
			}
			var response dnsmessage.Message
			if err := response.Unpack(responseWire); err != nil {
				t.Fatalf("unpack response: %v", err)
			}
			if response.ID != 0x1234 || !response.Response || !response.Authoritative {
				t.Fatalf("unexpected response header: %#v", response.Header)
			}
			if !response.RecursionDesired || response.RecursionAvailable {
				t.Fatalf("unexpected recursion flags: %#v", response.Header)
			}
			if len(response.Questions) != 1 || len(response.Answers) != test.wantAnswers {
				t.Fatalf("questions=%d answers=%d", len(response.Questions), len(response.Answers))
			}
			if test.wantAnswers == 0 {
				return
			}
			answer := response.Answers[0]
			if answer.Header.Name != response.Questions[0].Name || answer.Header.Class != dnsmessage.ClassINET || answer.Header.TTL != captiveDNSTTL {
				t.Fatalf("unexpected answer header: %#v", answer.Header)
			}
			a, ok := answer.Body.(*dnsmessage.AResource)
			if !ok || a.A != [4]byte{192, 168, 12, 1} {
				t.Fatalf("unexpected A answer: %#v", answer.Body)
			}
		})
	}
}

func TestBuildCaptiveDNSResponseDropsUnsafeQueries(t *testing.T) {
	question := dnsmessage.Question{
		Name:  dnsmessage.MustNewName("probe.example."),
		Type:  dnsmessage.TypeA,
		Class: dnsmessage.ClassINET,
	}
	cases := []struct {
		name  string
		query []byte
	}{
		{name: "malformed", query: []byte{0, 1, 2}},
		{name: "response", query: captiveDNSQuery(t, dnsmessage.Header{Response: true}, []dnsmessage.Question{question})},
		{name: "non query operation", query: captiveDNSQuery(t, dnsmessage.Header{OpCode: 1}, []dnsmessage.Question{question})},
		{name: "multiple questions", query: captiveDNSQuery(t, dnsmessage.Header{}, []dnsmessage.Question{question, question})},
		{name: "non IN class", query: captiveDNSQuery(t, dnsmessage.Header{}, []dnsmessage.Question{{
			Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassCHAOS,
		}})},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if response, ok := buildCaptiveDNSResponse(test.query); ok || response != nil {
				t.Fatal("unsafe query was answered")
			}
		})
	}
}

func TestCaptiveDNSUDPAndTCPUseBoundedDNSMessages(t *testing.T) {
	query := captiveDNSQuery(t, dnsmessage.Header{ID: 33}, []dnsmessage.Question{{
		Name:  dnsmessage.MustNewName("probe.example."),
		Type:  dnsmessage.TypeA,
		Class: dnsmessage.ClassINET,
	}})

	t.Run("UDP", func(t *testing.T) {
		server := newCaptiveDNSPacketConn(query)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- serveCaptiveDNSUDP(ctx, server) }()

		var response []byte
		select {
		case response = <-server.responses:
		case <-time.After(time.Second):
			t.Fatal("UDP listener did not return a response")
		}
		if len(response) == 0 || len(response) > captiveDNSMaxMessageBytes {
			t.Fatal("UDP listener did not return a bounded response")
		}

		cancel()
		_ = server.Close()
		if err := <-done; err != nil {
			t.Fatalf("UDP server stopped with error: %v", err)
		}
	})

	t.Run("TCP", func(t *testing.T) {
		server, client := net.Pipe()
		done := make(chan error, 1)
		go func() { done <- serveCaptiveDNSTCPConnection(server) }()

		frame := make([]byte, 2+len(query))
		binary.BigEndian.PutUint16(frame[:2], uint16(len(query)))
		copy(frame[2:], query)
		if _, err := client.Write(frame); err != nil {
			t.Fatalf("write TCP query: %v", err)
		}
		var responseLength [2]byte
		if _, err := io.ReadFull(client, responseLength[:]); err != nil {
			t.Fatalf("read TCP response length: %v", err)
		}
		response := make([]byte, binary.BigEndian.Uint16(responseLength[:]))
		if _, err := io.ReadFull(client, response); err != nil {
			t.Fatalf("read TCP response: %v", err)
		}
		if len(response) == 0 || len(response) > captiveDNSMaxMessageBytes {
			t.Fatalf("unexpected TCP response size %d", len(response))
		}
		_ = client.Close()
		if err := <-done; err != nil {
			t.Fatalf("TCP server: %v", err)
		}
	})

	t.Run("oversized TCP frame is rejected", func(t *testing.T) {
		server, client := net.Pipe()
		done := make(chan error, 1)
		go func() { done <- serveCaptiveDNSTCPConnection(server) }()
		var length [2]byte
		binary.BigEndian.PutUint16(length[:], captiveDNSMaxMessageBytes+1)
		if _, err := client.Write(length[:]); err != nil {
			t.Fatalf("write oversized length: %v", err)
		}
		_ = client.Close()
		if err := <-done; err == nil {
			t.Fatal("oversized TCP frame was accepted")
		}
	})
}

func TestCaptiveDNSListenConfigBindsOnlyProvidedInterface(t *testing.T) {
	original := captiveDNSBindToDevice
	t.Cleanup(func() { captiveDNSBindToDevice = original })

	var gotFD int
	var gotInterface string
	captiveDNSBindToDevice = func(fd int, interfaceName string) error {
		gotFD = fd
		gotInterface = interfaceName
		return nil
	}
	config := captiveDNSListenConfig("wlan0")
	if err := config.Control("udp4", apPortalGateway+":53", captiveDNSRawConn{fd: 17}); err != nil {
		t.Fatalf("run socket control: %v", err)
	}
	if gotFD != 17 || gotInterface != "wlan0" {
		t.Fatalf("socket bound to fd=%d interface=%q", gotFD, gotInterface)
	}
}

func TestCaptiveDNSPIDRecordSafetyAndLifecycle(t *testing.T) {
	t.Run("status accepts only the managed private record", func(t *testing.T) {
		record := captiveDNSPIDRecord{pid: 101, startTime: 77}
		withCaptiveDNSPIDTestSeams(t, record, captiveDNSProcessIdentity{
			startTime: 77,
			args:      []string{"/kvmapp/server/NanoKVM-Server", captiveDNSCommand, captiveDNSServeCommand},
		}, true)
		if err := captiveDNSStatus(); err != nil {
			t.Fatalf("status: %v", err)
		}
		if err := os.Chmod(captiveDNSPIDFile, 0o644); err != nil {
			t.Fatalf("chmod PID fixture: %v", err)
		}
		if err := captiveDNSStatus(); err == nil {
			t.Fatal("insecure PID record was accepted")
		}
	})

	t.Run("stop refuses a mismatched process", func(t *testing.T) {
		record := captiveDNSPIDRecord{pid: 102, startTime: 78}
		withCaptiveDNSPIDTestSeams(t, record, captiveDNSProcessIdentity{
			startTime: 78,
			args:      []string{"/bin/unrelated", "worker"},
		}, true)
		called := false
		captiveDNSSignalProcess = func(int, syscall.Signal) error {
			called = true
			return nil
		}
		if err := stopCaptiveDNS(); !errors.Is(err, errCaptiveDNSPIDMismatch) {
			t.Fatalf("stop mismatch error = %v", err)
		}
		if called {
			t.Fatal("stop signalled a mismatched process")
		}
	})

	t.Run("stop removes a dead managed record", func(t *testing.T) {
		record := captiveDNSPIDRecord{pid: 103, startTime: 79}
		withCaptiveDNSPIDTestSeams(t, record, captiveDNSProcessIdentity{}, false)
		if err := stopCaptiveDNS(); err != nil {
			t.Fatalf("stop dead record: %v", err)
		}
		if _, err := os.Lstat(captiveDNSPIDFile); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("dead PID record remains: %v", err)
		}
	})

	t.Run("symlink is rejected without following it", func(t *testing.T) {
		record := captiveDNSPIDRecord{pid: 104, startTime: 80}
		withCaptiveDNSPIDTestSeams(t, record, captiveDNSProcessIdentity{}, false)
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, []byte(formatCaptiveDNSPIDRecord(record)), 0o600); err != nil {
			t.Fatalf("write symlink target: %v", err)
		}
		if err := os.Remove(captiveDNSPIDFile); err != nil {
			t.Fatalf("remove PID fixture: %v", err)
		}
		if err := os.Symlink(target, captiveDNSPIDFile); err != nil {
			t.Fatalf("create PID symlink: %v", err)
		}
		if err := stopCaptiveDNS(); err == nil {
			t.Fatal("symlink PID record was accepted")
		}
	})
}

func TestWriteCaptiveDNSPIDRecordUsesPrivateFormat(t *testing.T) {
	originalPath := captiveDNSPIDFile
	originalOwner := captiveDNSPIDOwner
	originalInspector := captiveDNSInspectProcess
	originalSync := captiveDNSSyncPIDFile
	t.Cleanup(func() {
		captiveDNSPIDFile = originalPath
		captiveDNSPIDOwner = originalOwner
		captiveDNSInspectProcess = originalInspector
		captiveDNSSyncPIDFile = originalSync
	})

	captiveDNSPIDFile = filepath.Join(t.TempDir(), "runtime", "captive-dns.pid")
	captiveDNSPIDOwner = func(os.FileInfo) bool { return true }
	captiveDNSInspectProcess = func(pid int) (captiveDNSProcessIdentity, bool, error) {
		if pid != os.Getpid() {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return captiveDNSProcessIdentity{startTime: 123}, true, nil
	}
	record, err := writeCaptiveDNSPIDRecord()
	if err != nil {
		t.Fatalf("write PID record: %v", err)
	}
	if record.pid != os.Getpid() || record.startTime != 123 {
		t.Fatalf("unexpected PID record: %#v", record)
	}
	info, err := os.Lstat(captiveDNSPIDFile)
	if err != nil {
		t.Fatalf("stat PID record: %v", err)
	}
	if info.Mode().Perm() != 0o600 || !info.Mode().IsRegular() {
		t.Fatalf("PID record mode = %v", info.Mode())
	}
}

func TestWriteCaptiveDNSPIDRecordRemovesPartialRecordOnFailure(t *testing.T) {
	originalPath := captiveDNSPIDFile
	originalOwner := captiveDNSPIDOwner
	originalInspector := captiveDNSInspectProcess
	originalSync := captiveDNSSyncPIDFile
	t.Cleanup(func() {
		captiveDNSPIDFile = originalPath
		captiveDNSPIDOwner = originalOwner
		captiveDNSInspectProcess = originalInspector
		captiveDNSSyncPIDFile = originalSync
	})

	captiveDNSPIDFile = filepath.Join(t.TempDir(), "runtime", "captive-dns.pid")
	captiveDNSPIDOwner = func(os.FileInfo) bool { return true }
	captiveDNSInspectProcess = func(pid int) (captiveDNSProcessIdentity, bool, error) {
		if pid != os.Getpid() {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return captiveDNSProcessIdentity{startTime: 123}, true, nil
	}
	captiveDNSSyncPIDFile = func(*os.File) error { return errors.New("injected PID sync failure") }

	if _, err := writeCaptiveDNSPIDRecord(); err == nil {
		t.Fatal("PID record write unexpectedly succeeded")
	}
	if _, err := os.Lstat(captiveDNSPIDFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial PID record remains: %v", err)
	}
}

func TestCaptiveDNSProcessIdentityParsing(t *testing.T) {
	t.Run("process start time uses field 22 after parenthesized name", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "stat")
		fields := make([]string, 20)
		for index := range fields {
			fields[index] = "1"
		}
		fields[19] = "987654"
		if err := os.WriteFile(path, []byte("321 (server worker) "+strings.Join(fields, " ")+"\n"), 0o600); err != nil {
			t.Fatalf("write stat fixture: %v", err)
		}
		startTime, err := readCaptiveDNSProcessStartTime(path)
		if err != nil || startTime != 987654 {
			t.Fatalf("start time=%d err=%v", startTime, err)
		}
	})

	t.Run("command line requires the fixed private mode", func(t *testing.T) {
		identity := captiveDNSProcessIdentity{
			startTime: 11,
			args:      parseCaptiveDNSCommandLine([]byte("/kvmapp/server/NanoKVM-Server\x00captive-dns\x00serve\x00")),
		}
		if !captiveDNSProcessMatchesRecord(identity, captiveDNSPIDRecord{pid: 22, startTime: 11}) {
			t.Fatal("fixed captive DNS command was rejected")
		}
		identity.args[2] = captiveDNSStatusCommand
		if captiveDNSProcessMatchesRecord(identity, captiveDNSPIDRecord{pid: 22, startTime: 11}) {
			t.Fatal("status command was accepted as a daemon")
		}
	})
}

func TestRunCaptiveDNSCommandAcceptsOnlyFixedModes(t *testing.T) {
	if handled, _ := runCaptiveDNSCommand([]string{"not-captive-dns"}); handled {
		t.Fatal("unrelated command was intercepted")
	}
	for _, args := range [][]string{
		{captiveDNSCommand},
		{captiveDNSCommand, captiveDNSServeCommand, "unexpected"},
		{captiveDNSCommand, "custom-address"},
	} {
		if handled, code := runCaptiveDNSCommand(args); !handled || code != captiveDNSCommandExitUsage {
			t.Fatalf("args %q: handled=%t code=%d", args, handled, code)
		}
	}
}

func captiveDNSQuery(t *testing.T, header dnsmessage.Header, questions []dnsmessage.Question) []byte {
	t.Helper()
	message := dnsmessage.Message{Header: header, Questions: questions}
	query, err := message.Pack()
	if err != nil {
		t.Fatalf("pack query: %v", err)
	}
	return query
}

func withCaptiveDNSPIDTestSeams(t *testing.T, record captiveDNSPIDRecord, identity captiveDNSProcessIdentity, running bool) {
	t.Helper()
	originalPath := captiveDNSPIDFile
	originalOwner := captiveDNSPIDOwner
	originalInspector := captiveDNSInspectProcess
	originalSignal := captiveDNSSignalProcess
	t.Cleanup(func() {
		captiveDNSPIDFile = originalPath
		captiveDNSPIDOwner = originalOwner
		captiveDNSInspectProcess = originalInspector
		captiveDNSSignalProcess = originalSignal
	})

	captiveDNSPIDFile = filepath.Join(t.TempDir(), "captive-dns.pid")
	if err := os.WriteFile(captiveDNSPIDFile, []byte(formatCaptiveDNSPIDRecord(record)), 0o600); err != nil {
		t.Fatalf("write PID fixture: %v", err)
	}
	captiveDNSPIDOwner = func(os.FileInfo) bool { return true }
	captiveDNSInspectProcess = func(pid int) (captiveDNSProcessIdentity, bool, error) {
		if pid != record.pid {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return identity, running, nil
	}
	captiveDNSSignalProcess = func(int, syscall.Signal) error { return nil }
}

type captiveDNSRawConn struct {
	fd uintptr
}

func (connection captiveDNSRawConn) Control(callback func(uintptr)) error {
	callback(connection.fd)
	return nil
}

func (captiveDNSRawConn) Read(func(uintptr) bool) error {
	return nil
}

func (captiveDNSRawConn) Write(func(uintptr) bool) error {
	return nil
}

type captiveDNSPacketConn struct {
	query     []byte
	remote    net.Addr
	responses chan []byte
	closed    chan struct{}
	closeOnce sync.Once
	readOnce  bool
}

func newCaptiveDNSPacketConn(query []byte) *captiveDNSPacketConn {
	return &captiveDNSPacketConn{
		query:     append([]byte(nil), query...),
		remote:    &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 12345},
		responses: make(chan []byte, 1),
		closed:    make(chan struct{}),
	}
}

func (connection *captiveDNSPacketConn) ReadFrom(buffer []byte) (int, net.Addr, error) {
	if !connection.readOnce {
		connection.readOnce = true
		return copy(buffer, connection.query), connection.remote, nil
	}
	<-connection.closed
	return 0, nil, net.ErrClosed
}

func (connection *captiveDNSPacketConn) WriteTo(payload []byte, _ net.Addr) (int, error) {
	connection.responses <- append([]byte(nil), payload...)
	return len(payload), nil
}

func (connection *captiveDNSPacketConn) Close() error {
	connection.closeOnce.Do(func() { close(connection.closed) })
	return nil
}

func (connection *captiveDNSPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(192, 168, 12, 1), Port: captiveDNSPort}
}

func (*captiveDNSPacketConn) SetDeadline(time.Time) error {
	return nil
}

func (*captiveDNSPacketConn) SetReadDeadline(time.Time) error {
	return nil
}

func (*captiveDNSPacketConn) SetWriteDeadline(time.Time) error {
	return nil
}
