package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/service/authsession"
	"NanoKVM-Server/utils"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	messageWait     = 10 * time.Second
	maxMessageSize  = 1024
	TeminalAuthLock = "/etc/kvm/terminal_auth"
	terminalAuthDir = TeminalAuthLock + ".d"
	terminalAuthTTL = time.Minute
	terminalBash    = "/bin/bash"
	terminalSh      = "/bin/sh"
)

const terminalDefaultHome = "/root"

var terminalAuthGrantDir = terminalAuthDir

type terminalAuthGrant struct {
	Subject   string `json:"subject"`
	ExpiresAt int64  `json:"expiresAt"`
}

type WinSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
	CheckOrigin:     utils.CheckWebSocketOrigin,
}

var (
	terminalAuthenticationEnabled = func() bool {
		return config.GetInstance().Authentication != "disable"
	}
	verifyTerminalChallenge = authservice.VerifySensitiveActionChallenge
)

func (s *Service) TerminalAuth(ctx *gin.Context) {
	var rsp proto.Response
	if !terminalAuthenticationEnabled() {
		rsp.ErrRsp(ctx, -1, "authentication failed")
		return
	}
	if _, ok := terminalAuthSubject(ctx); !ok {
		rsp.ErrRsp(ctx, -1, "authentication failed")
		return
	}

	var req proto.SensitiveChallengeReq
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Challenge) == "" {
		rsp.ErrRsp(ctx, -1, "authentication failed")
		return
	}
	if err := verifyTerminalChallenge(ctx, req.Challenge, authservice.SensitiveActionOpenWebTerminal); err != nil {
		rsp.ErrRsp(ctx, -1, "authentication failed")
		return
	}

	if err := createTerminalAuthGrant(ctx, terminalAuthGrantDir, time.Now().Add(terminalAuthTTL)); err != nil {
		log.Printf("failed to create terminal auth grant")
		rsp.ErrRsp(ctx, -1, "authentication failed")
		return
	}
	rsp.OkRsp(ctx)
}

func createTerminalAuthGrant(ctx *gin.Context, dir string, expiresAt time.Time) error {
	subject, ok := terminalAuthSubject(ctx)
	if !ok {
		return os.ErrPermission
	}

	path := terminalAuthGrantPath(dir, subject)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}

	return json.NewEncoder(file).Encode(terminalAuthGrant{
		Subject:   subject,
		ExpiresAt: expiresAt.Unix(),
	})
}

func consumeTerminalAuthGrant(ctx *gin.Context, dir string, now time.Time) bool {
	subject, ok := terminalAuthSubject(ctx)
	if !ok {
		return false
	}

	path := terminalAuthGrantPath(dir, subject)
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var grant terminalAuthGrant
	if err := json.Unmarshal(content, &grant); err != nil {
		_ = os.Remove(path)
		return false
	}

	if grant.Subject == "" || grant.Subject != subject || grant.ExpiresAt <= now.Unix() {
		if grant.ExpiresAt <= now.Unix() {
			_ = os.Remove(path)
		}
		return false
	}

	if err := os.Remove(path); err != nil {
		log.Printf("failed to consume terminal auth grant: %v", err)
		return false
	}

	return true
}

func terminalAuthGrantPath(dir string, subject string) string {
	sum := sha256.Sum256([]byte(subject))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".json")
}

func terminalAuthSubject(ctx *gin.Context) (string, bool) {
	if ctx == nil || ctx.Request == nil {
		return "", false
	}

	username := strings.TrimSpace(ctx.GetString("username"))
	rawToken, ok := ctx.Get("jwtToken")
	if username == "" || !ok {
		return "", false
	}
	token, ok := rawToken.(string)
	token = strings.TrimSpace(token)
	if !ok || token == "" {
		return "", false
	}

	claims, err := middleware.ParseJWT(token)
	if err != nil || strings.TrimSpace(claims.Username) != username {
		return "", false
	}

	sum := sha256.Sum256([]byte(token))
	return "jwt:" + hex.EncodeToString(sum[:]), true
}

func (s *Service) Terminal(c *gin.Context) {
	if !terminalAuthenticationEnabled() {
		log.Printf("terminal authorization unavailable")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if _, ok := terminalAuthSubject(c); !ok {
		log.Printf("terminal authorization identity missing")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if !consumeTerminalAuthGrant(c, terminalAuthGrantDir, time.Now()) {
		log.Printf("terminal secondary authorization failed")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("failed to init websocket: %s", err)
		return
	}
	defer func() {
		_ = ws.Close()
	}()
	unregister := authsession.Register(func(code int, text string) {
		_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(2*time.Second))
		_ = ws.Close()
	})
	defer unregister()

	cmd := exec.Command(terminalShell())
	cmd.Env = terminalEnvironment(os.Environ(), terminalDefaultHome)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Errorf("failed to start pty: %s", err)
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		// Wait & reap sub-process; needed to prevent zombie processes
		_ = cmd.Wait()
	}()

	go wsWrite(ws, ptmx)
	wsRead(ws, ptmx)
}

func terminalShell() string {
	if info, err := os.Stat(terminalBash); err == nil && !info.IsDir() {
		return terminalBash
	}

	return terminalSh
}

func terminalEnvironment(env []string, fallbackHome string) []string {
	if environmentValue(env, "HOME") == "" {
		// System services do not get a login environment, so Bash may have no HOME.
		env = append(env, "HOME="+fallbackHome)
	}

	return append(env,
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
}

func environmentValue(env []string, key string) string {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix)
		}
	}

	return ""
}

// pty to ws
func wsWrite(ws *websocket.Conn, ptmx *os.File) {
	data := make([]byte, maxMessageSize)

	for {
		n, err := ptmx.Read(data)
		if err != nil {
			return
		}

		if n > 0 {
			_ = ws.SetWriteDeadline(time.Now().Add(messageWait))

			err = ws.WriteMessage(websocket.BinaryMessage, data[:n])
			if err != nil {
				log.Errorf("write ws message failed: %s", err)
				return
			}
		}
	}
}

// ws to pty
func wsRead(ws *websocket.Conn, ptmx *os.File) {
	var zeroTime time.Time
	_ = ws.SetReadDeadline(zeroTime)

	for {
		msgType, p, err := ws.ReadMessage()
		if err != nil {
			return
		}

		// resize message
		if msgType == websocket.BinaryMessage {
			var winSize WinSize
			if err := json.Unmarshal(p, &winSize); err == nil {
				_ = pty.Setsize(ptmx, &pty.Winsize{
					Rows: winSize.Rows,
					Cols: winSize.Cols,
				})
			}
			continue
		}

		_, err = ptmx.Write(p)
		if err != nil {
			log.Errorf("failed to write to pty: %s", err)
			return
		}
	}
}
