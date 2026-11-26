package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/config"
	"NanoKVM-Server/logger"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/router"
	"NanoKVM-Server/service/ai"
	"NanoKVM-Server/service/application"
	"NanoKVM-Server/service/storage/diskfs"
	"NanoKVM-Server/service/uistate"
	"NanoKVM-Server/service/vm"
	"NanoKVM-Server/service/vm/jiggler"

	"github.com/gin-gonic/gin"
)

const (
	publicReadHeaderTimeout   = 10 * time.Second
	publicIdleTimeout         = 2 * time.Minute
	redirectReadHeaderTimeout = 5 * time.Second
	redirectReadTimeout       = 10 * time.Second
	redirectWriteTimeout      = 10 * time.Second
	redirectIdleTimeout       = 30 * time.Second
	serverMaxHeaderBytes      = 64 << 10
	apPortalGateway           = "192.168.12.1"
	apPortalHTTPPort          = 80
	apPortalPasswordMaxBytes  = 63
)

var (
	apPortalGatewayIP    = net.ParseIP(apPortalGateway)
	apPortalPasswordFile = "/tmp/ap.pass"
	uiStateSocketStart   = uistate.StartServer
	uiStateSocketStop    = uistate.StopServer
)

func main() {
	if handled, code := application.RunUpdateWorkerCommand(os.Args[1:]); handled {
		os.Exit(code)
	}
	if handled, code := runCaptiveDNSCommand(os.Args[1:]); handled {
		os.Exit(code)
	}
	os.Exit(runServer())
}

func runServer() int {
	initialize()
	defer dispose()

	if err := run(); err != nil {
		log.Printf("HTTP server stopped: %v", err)
		return 1
	}
	return 0
}

func initialize() {
	logger.Init()
	diskfs.CleanupStale()

	if err := ai.StartStateSocketServer(); err != nil {
		log.Fatalf("state socket server failed: %v", err)
	}
	if err := ai.CleanupScreenTimelapseTombstones(); err != nil {
		log.Printf("failed to clean leftover screen timelapse tombstones: %v", err)
	}
	ai.StartScreenTimelapseRetentionWorker()

	// sync HDMI hardware with persisted capture state
	syncHDMICaptureState()
	if err := vm.RestoreLegacyEdid(); err != nil {
		log.Printf("failed to restore the selected EDID profile on upgrade: %v", err)
	}

	// init screen parameters
	_ = common.GetScreen()

	// run mouse jiggler
	jiggler.GetJiggler()

	// Register shutdown handling before PWM initialization, which may need to
	// wait for device nodes or an external devmem command.
	watchForExitSignal()

	initializeFingerRobotIfEnabled()
}

var (
	fingerRobotProfilesValidate     = vm.ValidateFingerRobotArmProfiles
	fingerRobotSettingEnabled       = vm.IsFingerRobotSettingEnabled
	fingerRobotControllerInitialize = vm.InitializeFingerRobotController
)

func initializeFingerRobotIfEnabled() {
	if err := fingerRobotProfilesValidate(); err != nil {
		log.Printf("invalid finger robot arm profiles: %v", err)
		return
	}

	if !fingerRobotSettingEnabled() {
		return
	}

	// The server remains available if PWM hardware is absent, but all finger
	// robot actions fail closed until the controller is initialized.
	if err := fingerRobotControllerInitialize(); err != nil {
		log.Printf("failed to initialize finger robot controller: %v", err)
	}
}

func watchForExitSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigChan
		log.Printf("\nReceived signal: %v\n", sig)

		dispose()
		os.Exit(0)
	}()
}

func syncHDMICaptureState() {
	enabled := config.GetHDMICaptureEnabled()
	if result := common.GetKvmVision().SetHDMI(enabled); result < 0 {
		log.Printf("failed to sync HDMI capture status to %t: %d", enabled, result)
	}
}

func run() error {
	conf := config.GetInstance()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	if conf.Proto != "http" {
		r.Use(middleware.Tls())
	}
	r.Use(middleware.CSRFProtection())

	if conf.Authentication == "disable" {
		r.Use(middleware.DevelopmentCORS(conf.Security.CORS.AllowedOrigins))
	}

	router.Init(r)
	startUIStateSocketServer()
	httpAddr := fmt.Sprintf(":%d", conf.Port.Http)

	if conf.Proto == "http" {
		log.Printf("Starting HTTP server on %s\n", httpAddr)
		return newPublicHTTPServer(httpAddr, r).ListenAndServe()
	} else {
		httpsAddr := fmt.Sprintf(":%d", conf.Port.Https)
		log.Printf("Starting HTTPS server on %s, %s\n", httpAddr, httpsAddr)

		go func() {
			if err := runRedirect(httpAddr, httpsAddr, r); err != nil {
				log.Printf("HTTP redirect server stopped: %v", err)
			}
		}()

		return newPublicHTTPServer(httpsAddr, r).ListenAndServeTLS(conf.Cert.Crt, conf.Cert.Key)
	}
}

func startUIStateSocketServer() {
	if err := uiStateSocketStart(); err != nil {
		log.Printf("failed to start UI status socket server: %v", err)
	}
}

func stopUIStateSocketServer() {
	if err := uiStateSocketStop(); err != nil {
		log.Printf("failed to stop UI status socket server: %v", err)
	}
}

func newPublicHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: publicReadHeaderTimeout,
		IdleTimeout:       publicIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
		// ReadTimeout and WriteTimeout intentionally remain zero: this server
		// owns long-lived SSE, WebSocket, and media streams. Finite handlers
		// enforce their own body and operation limits.
	}
}

func runRedirect(httpPort string, httpsPort string, applicationHandler http.Handler) error {
	return newRedirectHTTPServer(httpPort, redirectHandler(httpsPort, applicationHandler)).ListenAndServe()
}

func redirectHandler(httpsPort string, applicationHandler http.Handler) http.Handler {
	redirectRouter := gin.New()
	redirectRouter.Use(middleware.SecurityHeaders())
	redirectRouter.NoRoute(func(c *gin.Context) {
		if isAPPortalLocalRequest(c.Request) {
			if !isAPPortalGatewayHost(c.Request.Host) {
				http.Redirect(c.Writer, c.Request, apPortalRedirectTarget(), http.StatusFound)
				return
			}

			if applicationHandler != nil && isAPPortalApplicationRequest(c.Request) {
				applicationHandler.ServeHTTP(c.Writer, c.Request)
				return
			}

			c.Status(http.StatusNotFound)
			return
		}

		http.Redirect(c.Writer, c.Request, redirectTarget(c.Request, httpsPort), http.StatusTemporaryRedirect)
	})
	return redirectRouter
}

func isAPPortalLocalRequest(req *http.Request) bool {
	addr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || addr == nil {
		return false
	}

	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		return tcpAddr.Port == apPortalHTTPPort && tcpAddr.IP != nil && tcpAddr.IP.Equal(apPortalGatewayIP)
	}

	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	return port == "80" && ip != nil && ip.Equal(apPortalGatewayIP)
}

func isAPPortalGatewayHost(host string) bool {
	return strings.TrimSuffix(normalizedRedirectRequestHost(host), ".") == apPortalGateway
}

func isAPPortalApplicationRequest(req *http.Request) bool {
	path := req.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return (req.Method == http.MethodPost && (path == "/api/network/wifi/verify" || path == "/api/network/wifi")) ||
			(req.Method == http.MethodGet && path == "/api/network/wifi/scan")
	}

	return req.Method == http.MethodGet || req.Method == http.MethodHead
}

func apPortalRedirectTarget() string {
	password, err := readBoundedAPPortalPassword()
	if err != nil || password == "" {
		return "http://" + apPortalGateway + "/#/wifi"
	}

	encodedPassword := url.Values{"p": []string{password}}.Encode()
	return "http://" + apPortalGateway + "/#/wifi?" + encodedPassword
}

func readBoundedAPPortalPassword() (string, error) {
	file, err := os.OpenFile(apPortalPasswordFile, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("invalid AP password file")
	}

	contents, err := io.ReadAll(io.LimitReader(file, apPortalPasswordMaxBytes+3))
	if err != nil {
		return "", err
	}
	if len(contents) > apPortalPasswordMaxBytes+2 {
		return "", errors.New("AP password exceeds maximum length")
	}

	password := strings.TrimSuffix(string(contents), "\n")
	password = strings.TrimSuffix(password, "\r")
	if password == "" || len(password) > apPortalPasswordMaxBytes || strings.ContainsAny(password, "\r\n") {
		return "", errors.New("invalid AP password")
	}

	return password, nil
}

func newRedirectHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: redirectReadHeaderTimeout,
		ReadTimeout:       redirectReadTimeout,
		WriteTimeout:      redirectWriteTimeout,
		IdleTimeout:       redirectIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}
}

func redirectTarget(req *http.Request, httpsPort string) string {
	host := redirectHost(req)
	port := strings.TrimPrefix(httpsPort, ":")
	if port == "" {
		port = "443"
	}

	targetHost := formatRedirectHost(host, port)
	target := url.URL{
		Scheme:   "https",
		Host:     targetHost,
		Path:     req.URL.Path,
		RawPath:  req.URL.RawPath,
		RawQuery: req.URL.RawQuery,
	}
	if target.Path == "" {
		target.Path = "/"
	}

	return target.String()
}

func redirectHost(req *http.Request) string {
	if addr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr); ok && addr != nil {
		if tcpAddr, ok := addr.(*net.TCPAddr); ok && tcpAddr.IP != nil && !tcpAddr.IP.IsUnspecified() {
			return tcpAddr.IP.String()
		}
		if host, _, err := net.SplitHostPort(addr.String()); err == nil && host != "" && host != "::" && host != "0.0.0.0" {
			return host
		}
	}

	if host := normalizedRedirectRequestHost(req.Host); host != "" {
		if isAllowedRedirectRequestHost(host) {
			return host
		}
	}

	return "localhost"
}

var redirectTrustedHosts = func() []string {
	return config.GetInstance().Security.Redirect.TrustedHosts
}

func normalizedRedirectRequestHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "/\\@") {
		return ""
	}
	for _, char := range host {
		if char <= 31 || char == 127 {
			return ""
		}
	}

	if strings.HasPrefix(host, "[") {
		if value, _, err := net.SplitHostPort(host); err == nil && value != "" {
			return value
		}
		if strings.HasSuffix(host, "]") {
			value := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
			if value != "" {
				return value
			}
		}
		return ""
	}

	if value, _, err := net.SplitHostPort(host); err == nil && value != "" {
		return value
	}

	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return ""
	}

	return host
}

func isAllowedRedirectRequestHost(host string) bool {
	if isLocalRedirectHost(host) {
		return true
	}

	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	for _, trusted := range redirectTrustedHosts() {
		trusted = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(trusted), "."))
		if trusted != "" && host == trusted {
			return true
		}
	}

	return false
}

func isLocalRedirectHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") || !strings.Contains(host, ".") {
		return true
	}

	return false
}

func formatRedirectHost(host string, port string) string {
	if port != "443" {
		return net.JoinHostPort(host, port)
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

var disposeOnce sync.Once

func dispose() {
	disposeOnce.Do(func() {
		if err := vm.ShutdownFingerRobotController(); err != nil {
			log.Printf("failed to release finger robot during shutdown: %v", err)
		}
		ai.StopScreenTimelapseRetentionWorker()
		if err := ai.StopStateSocketServer(); err != nil {
			log.Printf("failed to stop state socket server: %v", err)
		}
		stopUIStateSocketServer()
		diskfs.CleanupStale()
		common.GetKvmVision().Close()
	})
}
