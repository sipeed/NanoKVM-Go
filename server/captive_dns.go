package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	captiveDNSCommand       = "captive-dns"
	captiveDNSServeCommand  = "serve"
	captiveDNSStatusCommand = "status"
	captiveDNSStopCommand   = "stop"

	captiveDNSPort               = 53
	captiveDNSPIDFilePath        = "/run/nanokvm/captive-dns.pid"
	captiveDNSMaxMessageBytes    = 1232
	captiveDNSPIDRecordMaxBytes  = 64
	captiveDNSTTL                = 30
	captiveDNSUDPReadTimeout     = 250 * time.Millisecond
	captiveDNSTCPTimeout         = 2 * time.Second
	captiveDNSMaxTCPConnections  = 4
	captiveDNSCommandExitFailure = 1
	captiveDNSCommandExitUsage   = 2
)

var (
	captiveDNSPIDFile = captiveDNSPIDFilePath

	captiveDNSFindGatewayInterface = findCaptiveDNSGatewayInterface
	captiveDNSBindListeners        = bindCaptiveDNSListeners
	captiveDNSInspectProcess       = inspectCaptiveDNSProcess
	captiveDNSSignalProcess        = signalCaptiveDNSProcess
	captiveDNSPIDOwner             = captiveDNSOwnedByRoot
	captiveDNSBindToDevice         = bindCaptiveDNSSocketToDevice
	captiveDNSSyncPIDFile          = func(file *os.File) error { return file.Sync() }
)

var (
	errCaptiveDNSNotRunning  = errors.New("AP captive DNS is not running")
	errCaptiveDNSPIDMismatch = errors.New("AP captive DNS PID record does not identify the expected process")
)

type captiveDNSListeners struct {
	udp net.PacketConn
	tcp net.Listener
}

func (listeners captiveDNSListeners) close() {
	if listeners.udp != nil {
		_ = listeners.udp.Close()
	}
	if listeners.tcp != nil {
		_ = listeners.tcp.Close()
	}
}

type captiveDNSPIDRecord struct {
	pid       int
	startTime uint64
}

type captiveDNSProcessIdentity struct {
	startTime uint64
	args      []string
}

func runCaptiveDNSCommand(args []string) (handled bool, code int) {
	if len(args) == 0 || args[0] != captiveDNSCommand {
		return false, 0
	}
	if len(args) != 2 {
		return true, captiveDNSCommandExitUsage
	}

	var err error
	switch args[1] {
	case captiveDNSServeCommand:
		err = serveCaptiveDNS()
	case captiveDNSStatusCommand:
		err = captiveDNSStatus()
	case captiveDNSStopCommand:
		err = stopCaptiveDNS()
	default:
		return true, captiveDNSCommandExitUsage
	}
	if err != nil {
		// Errors deliberately exclude DNS request names and AP credentials.
		log.Printf("AP captive DNS %s failed: %v", args[1], err)
		return true, captiveDNSCommandExitFailure
	}
	return true, 0
}

func serveCaptiveDNS() error {
	iface, err := captiveDNSFindGatewayInterface()
	if err != nil {
		return fmt.Errorf("find AP gateway interface: %w", err)
	}

	listeners, err := captiveDNSBindListeners(iface)
	if err != nil {
		return fmt.Errorf("bind AP captive DNS listeners: %w", err)
	}
	defer listeners.close()

	record, err := writeCaptiveDNSPIDRecord()
	if err != nil {
		return fmt.Errorf("write AP captive DNS PID record: %w", err)
	}
	defer func() {
		if err := removeCaptiveDNSPIDRecord(record); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("failed to remove AP captive DNS PID record: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serveCaptiveDNSListeners(ctx, listeners)
}

func findCaptiveDNSGatewayInterface() (net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}

	var matches []net.Interface
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			return net.Interface{}, err
		}
		for _, address := range addresses {
			if captiveDNSAddressMatchesGateway(address) {
				matches = append(matches, iface)
				break
			}
		}
	}
	if len(matches) != 1 {
		return net.Interface{}, fmt.Errorf("expected exactly one AP gateway interface, found %d", len(matches))
	}
	return matches[0], nil
}

func captiveDNSAddressMatchesGateway(address net.Addr) bool {
	switch value := address.(type) {
	case *net.IPNet:
		return value.IP.Equal(apPortalGatewayIP)
	case *net.IPAddr:
		return value.IP.Equal(apPortalGatewayIP)
	default:
		return false
	}
}

func bindCaptiveDNSListeners(iface net.Interface) (captiveDNSListeners, error) {
	if iface.Name == "" {
		return captiveDNSListeners{}, errors.New("empty AP gateway interface")
	}

	listenConfig := captiveDNSListenConfig(iface.Name)
	address := net.JoinHostPort(apPortalGateway, strconv.Itoa(captiveDNSPort))
	udp, err := listenConfig.ListenPacket(context.Background(), "udp4", address)
	if err != nil {
		return captiveDNSListeners{}, err
	}

	tcp, err := listenConfig.Listen(context.Background(), "tcp4", address)
	if err != nil {
		_ = udp.Close()
		return captiveDNSListeners{}, err
	}
	return captiveDNSListeners{udp: udp, tcp: tcp}, nil
}

func captiveDNSListenConfig(interfaceName string) net.ListenConfig {
	return net.ListenConfig{
		Control: func(_ string, _ string, rawConn syscall.RawConn) error {
			var socketErr error
			if err := rawConn.Control(func(fd uintptr) {
				socketErr = captiveDNSBindToDevice(int(fd), interfaceName)
			}); err != nil {
				return err
			}
			return socketErr
		},
	}
}

func bindCaptiveDNSSocketToDevice(fd int, interfaceName string) error {
	return syscall.SetsockoptString(fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, interfaceName)
}

func serveCaptiveDNSListeners(ctx context.Context, listeners captiveDNSListeners) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		listeners.close()
	}()

	errs := make(chan error, 2)
	go func() {
		errs <- serveCaptiveDNSUDP(ctx, listeners.udp)
	}()
	go func() {
		errs <- serveCaptiveDNSTCP(ctx, listeners.tcp)
	}()

	var firstErr error
	for range 2 {
		err := <-errs
		if err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	return firstErr
}

func serveCaptiveDNSUDP(ctx context.Context, connection net.PacketConn) error {
	if connection == nil {
		return errors.New("missing AP captive DNS UDP listener")
	}

	buffer := make([]byte, captiveDNSMaxMessageBytes+1)
	for {
		if err := connection.SetReadDeadline(time.Now().Add(captiveDNSUDPReadTimeout)); err != nil {
			return fmt.Errorf("set UDP DNS read deadline: %w", err)
		}
		n, remoteAddress, err := connection.ReadFrom(buffer)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return fmt.Errorf("read UDP DNS request: %w", err)
		}
		if n > captiveDNSMaxMessageBytes {
			continue
		}

		response, ok := buildCaptiveDNSResponse(buffer[:n])
		if !ok {
			continue
		}
		// A failed client write must not stop the AP DNS listener. The response
		// contains no client-controlled name in any log or error path.
		_ = connection.SetWriteDeadline(time.Now().Add(captiveDNSUDPReadTimeout))
		_, _ = connection.WriteTo(response, remoteAddress)
	}
}

func serveCaptiveDNSTCP(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return errors.New("missing AP captive DNS TCP listener")
	}

	semaphore := make(chan struct{}, captiveDNSMaxTCPConnections)
	var workers sync.WaitGroup
	defer workers.Wait()
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept TCP DNS connection: %w", err)
		}

		select {
		case semaphore <- struct{}{}:
			workers.Add(1)
			go func(connection net.Conn) {
				defer workers.Done()
				defer func() { <-semaphore }()
				serveCaptiveDNSTCPWorker(ctx, connection)
			}(connection)
		default:
			_ = connection.Close()
		}
	}
}

func serveCaptiveDNSTCPWorker(ctx context.Context, connection net.Conn) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-done:
		}
	}()
	_ = serveCaptiveDNSTCPConnection(connection)
	close(done)
}

func serveCaptiveDNSTCPConnection(connection net.Conn) error {
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(captiveDNSTCPTimeout)); err != nil {
		return err
	}

	var lengthBytes [2]byte
	if _, err := io.ReadFull(connection, lengthBytes[:]); err != nil {
		return err
	}
	length := int(lengthBytes[0])<<8 | int(lengthBytes[1])
	if length == 0 || length > captiveDNSMaxMessageBytes {
		return errors.New("invalid TCP DNS message length")
	}

	query := make([]byte, length)
	if _, err := io.ReadFull(connection, query); err != nil {
		return err
	}
	response, ok := buildCaptiveDNSResponse(query)
	if !ok {
		return nil
	}
	if len(response) > math.MaxUint16 {
		return errors.New("TCP DNS response exceeds framing limit")
	}

	responseLength := [2]byte{byte(len(response) >> 8), byte(len(response))}
	if _, err := connection.Write(responseLength[:]); err != nil {
		return err
	}
	_, err := connection.Write(response)
	return err
}

func buildCaptiveDNSResponse(packet []byte) ([]byte, bool) {
	if len(packet) == 0 || len(packet) > captiveDNSMaxMessageBytes {
		return nil, false
	}

	var query dnsmessage.Message
	if err := query.Unpack(packet); err != nil {
		return nil, false
	}
	if query.Response || query.OpCode != 0 || len(query.Questions) != 1 {
		return nil, false
	}
	question := query.Questions[0]
	if question.Class != dnsmessage.ClassINET {
		return nil, false
	}

	response := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:               query.ID,
			Response:         true,
			Authoritative:    true,
			RecursionDesired: query.RecursionDesired,
		},
		Questions: []dnsmessage.Question{question},
	}
	if question.Type == dnsmessage.TypeA {
		response.Answers = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{
				Name:  question.Name,
				Class: dnsmessage.ClassINET,
				TTL:   captiveDNSTTL,
			},
			Body: &dnsmessage.AResource{A: [4]byte{192, 168, 12, 1}},
		}}
	}

	packed, err := response.Pack()
	if err != nil || len(packed) > captiveDNSMaxMessageBytes {
		return nil, false
	}
	return packed, true
}

func captiveDNSStatus() error {
	record, err := readCaptiveDNSPIDRecord()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errCaptiveDNSNotRunning
		}
		return err
	}
	identity, running, err := captiveDNSInspectProcess(record.pid)
	if err != nil {
		return fmt.Errorf("inspect AP captive DNS process: %w", err)
	}
	if !running {
		return errCaptiveDNSNotRunning
	}
	if !captiveDNSProcessMatchesRecord(identity, record) {
		return errCaptiveDNSPIDMismatch
	}
	return nil
}

func stopCaptiveDNS() error {
	record, err := readCaptiveDNSPIDRecord()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	identity, running, err := captiveDNSInspectProcess(record.pid)
	if err != nil {
		return fmt.Errorf("inspect AP captive DNS process: %w", err)
	}
	if !running {
		return removeCaptiveDNSPIDRecord(record)
	}
	if !captiveDNSProcessMatchesRecord(identity, record) {
		return errCaptiveDNSPIDMismatch
	}
	if err := captiveDNSSignalProcess(record.pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("signal AP captive DNS process: %w", err)
	}
	return nil
}

func writeCaptiveDNSPIDRecord() (result captiveDNSPIDRecord, err error) {
	if err := reclaimDeadCaptiveDNSPIDRecord(); err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if err := ensureCaptiveDNSPIDDirectory(); err != nil {
		return captiveDNSPIDRecord{}, err
	}

	identity, running, err := captiveDNSInspectProcess(os.Getpid())
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if !running || identity.startTime == 0 {
		return captiveDNSPIDRecord{}, errors.New("cannot identify AP captive DNS process")
	}
	record := captiveDNSPIDRecord{pid: os.Getpid(), startTime: identity.startTime}

	file, err := os.OpenFile(captiveDNSPIDFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			// This file was created exclusively in a root-owned, non-writable
			// runtime directory. Do not leave a partial record that would make
			// later status/stop operations fail closed forever.
			_ = os.Remove(captiveDNSPIDFile)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if _, err := file.WriteString(formatCaptiveDNSPIDRecord(record)); err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if err := captiveDNSSyncPIDFile(file); err != nil {
		return captiveDNSPIDRecord{}, err
	}
	return record, nil
}

func reclaimDeadCaptiveDNSPIDRecord() error {
	record, err := readCaptiveDNSPIDRecord()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	identity, running, err := captiveDNSInspectProcess(record.pid)
	if err != nil {
		return err
	}
	if !running {
		return removeCaptiveDNSPIDRecord(record)
	}
	if !captiveDNSProcessMatchesRecord(identity, record) {
		return errCaptiveDNSPIDMismatch
	}
	return errors.New("AP captive DNS is already running")
}

func ensureCaptiveDNSPIDDirectory() error {
	directory := filepath.Dir(captiveDNSPIDFile)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 || !captiveDNSPIDOwner(info) {
		return errors.New("insecure AP captive DNS runtime directory")
	}
	return nil
}

func readCaptiveDNSPIDRecord() (captiveDNSPIDRecord, error) {
	pathInfo, err := os.Lstat(captiveDNSPIDFile)
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if err := validateCaptiveDNSPIDFileInfo(pathInfo); err != nil {
		return captiveDNSPIDRecord{}, err
	}

	file, err := os.OpenFile(captiveDNSPIDFile, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	defer file.Close()

	openedInfo, err := file.Stat()
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if !os.SameFile(pathInfo, openedInfo) {
		return captiveDNSPIDRecord{}, errors.New("AP captive DNS PID record changed while opening")
	}
	if err := validateCaptiveDNSPIDFileInfo(openedInfo); err != nil {
		return captiveDNSPIDRecord{}, err
	}

	contents, err := io.ReadAll(io.LimitReader(file, captiveDNSPIDRecordMaxBytes+1))
	if err != nil {
		return captiveDNSPIDRecord{}, err
	}
	if len(contents) > captiveDNSPIDRecordMaxBytes {
		return captiveDNSPIDRecord{}, errors.New("AP captive DNS PID record is too large")
	}
	return parseCaptiveDNSPIDRecord(contents)
}

func validateCaptiveDNSPIDFileInfo(info os.FileInfo) error {
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || !captiveDNSPIDOwner(info) {
		return errors.New("insecure AP captive DNS PID record")
	}
	return nil
}

func captiveDNSOwnedByRoot(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

func formatCaptiveDNSPIDRecord(record captiveDNSPIDRecord) string {
	return fmt.Sprintf("%d %d\n", record.pid, record.startTime)
}

func parseCaptiveDNSPIDRecord(contents []byte) (captiveDNSPIDRecord, error) {
	parts := strings.Fields(string(contents))
	if len(parts) != 2 {
		return captiveDNSPIDRecord{}, errors.New("invalid AP captive DNS PID record")
	}
	pid64, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || pid64 < 1 || pid64 > int64(^uint(0)>>1) {
		return captiveDNSPIDRecord{}, errors.New("invalid AP captive DNS PID")
	}
	startTime, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || startTime == 0 {
		return captiveDNSPIDRecord{}, errors.New("invalid AP captive DNS process start time")
	}
	record := captiveDNSPIDRecord{pid: int(pid64), startTime: startTime}
	if string(contents) != formatCaptiveDNSPIDRecord(record) {
		return captiveDNSPIDRecord{}, errors.New("invalid AP captive DNS PID record format")
	}
	return record, nil
}

func removeCaptiveDNSPIDRecord(expected captiveDNSPIDRecord) error {
	record, err := readCaptiveDNSPIDRecord()
	if err != nil {
		return err
	}
	if record != expected {
		return errors.New("AP captive DNS PID record changed before removal")
	}
	return os.Remove(captiveDNSPIDFile)
}

func inspectCaptiveDNSProcess(pid int) (captiveDNSProcessIdentity, bool, error) {
	if err := syscall.Kill(pid, 0); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return captiveDNSProcessIdentity{}, false, err
	}

	procPath := filepath.Join("/proc", strconv.Itoa(pid))
	commandLine, err := os.ReadFile(filepath.Join(procPath, "cmdline"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return captiveDNSProcessIdentity{}, false, err
	}
	startTime, err := readCaptiveDNSProcessStartTime(filepath.Join(procPath, "stat"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return captiveDNSProcessIdentity{}, false, nil
		}
		return captiveDNSProcessIdentity{}, false, err
	}

	return captiveDNSProcessIdentity{
		startTime: startTime,
		args:      parseCaptiveDNSCommandLine(commandLine),
	}, true, nil
}

func readCaptiveDNSProcessStartTime(path string) (uint64, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	closingParenthesis := strings.LastIndex(string(contents), ")")
	if closingParenthesis < 0 {
		return 0, errors.New("invalid AP captive DNS process stat")
	}
	fields := strings.Fields(string(contents[closingParenthesis+1:]))
	// /proc/<pid>/stat begins with pid (field 1) and comm (field 2). The
	// process start time is field 22, so it is index 19 after the final ')'.
	if len(fields) <= 19 {
		return 0, errors.New("incomplete AP captive DNS process stat")
	}
	return strconv.ParseUint(fields[19], 10, 64)
}

func parseCaptiveDNSCommandLine(commandLine []byte) []string {
	commandLine = bytes.TrimSuffix(commandLine, []byte{0})
	if len(commandLine) == 0 {
		return nil
	}
	parts := bytes.Split(commandLine, []byte{0})
	args := make([]string, len(parts))
	for index, part := range parts {
		args[index] = string(part)
	}
	return args
}

func captiveDNSProcessMatchesRecord(identity captiveDNSProcessIdentity, record captiveDNSPIDRecord) bool {
	return identity.startTime == record.startTime && len(identity.args) == 3 &&
		identity.args[1] == captiveDNSCommand && identity.args[2] == captiveDNSServeCommand
}

func signalCaptiveDNSProcess(pid int, signal syscall.Signal) error {
	return syscall.Kill(pid, signal)
}
