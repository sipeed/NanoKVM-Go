package frp

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"NanoKVM-Server/proto"

	"github.com/pelletier/go-toml/v2"
)

const adminPort = 17400

// The generated TOML is the only saved configuration. Only the four form
// fields are editable through the API; credentials never appear in responses.
type clientConfig struct {
	ServerAddr    string `toml:"serverAddr"`
	ServerPort    int    `toml:"serverPort"`
	LoginFailExit bool   `toml:"loginFailExit"`
	Auth          struct {
		Method string `toml:"method"`
		Token  string `toml:"token"`
	} `toml:"auth"`
	WebServer struct {
		Addr     string `toml:"addr"`
		Port     int    `toml:"port"`
		User     string `toml:"user"`
		Password string `toml:"password"`
	} `toml:"webServer"`
	Transport struct {
		TLS struct {
			Enable bool `toml:"enable"`
		} `toml:"tls"`
	} `toml:"transport"`
	Proxies []proxyConfig `toml:"proxies"`
}

type proxyConfig struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`
	LocalIP    string `toml:"localIP"`
	LocalPort  int    `toml:"localPort"`
	RemotePort int    `toml:"remotePort"`
}

func validateForm(req *proto.SetFRPConfigReq) error {
	req.ServerAddr = strings.TrimSpace(req.ServerAddr)
	// Bracketed IPv6 is accepted in the form, but frpc wants the bare address.
	if strings.HasPrefix(req.ServerAddr, "[") && strings.HasSuffix(req.ServerAddr, "]") {
		req.ServerAddr = strings.TrimSuffix(strings.TrimPrefix(req.ServerAddr, "["), "]")
	}
	if !validServerAddr(req.ServerAddr) {
		return errors.New("invalidServer")
	}
	if req.ServerPort < 1 || req.ServerPort > 65535 || req.RemotePort < 1 || req.RemotePort > 65535 || req.ServerPort == req.RemotePort {
		return errors.New("invalidPort")
	}
	// frpc renders Go templates before parsing TOML. Form tokens must stay
	// literal, rather than expanding environment variables or template helpers.
	if req.Token != nil && (len(*req.Token) > 4096 || !utf8.ValidString(*req.Token) || strings.ContainsAny(*req.Token, "\x00\r\n") || strings.Contains(*req.Token, "{{")) {
		return errors.New("invalidToken")
	}
	return nil
}

func validServerAddr(value string) bool {
	if net.ParseIP(value) != nil {
		return true
	}
	if len(value) == 0 || len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(value, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func makeConfig(req proto.SetFRPConfigReq, previous *clientConfig, localPort int) (*clientConfig, error) {
	cfg := &clientConfig{ServerAddr: req.ServerAddr, ServerPort: req.ServerPort, LoginFailExit: true}
	cfg.Auth.Method = "token"
	cfg.Transport.TLS.Enable = true
	cfg.WebServer.Addr = "127.0.0.1"
	cfg.WebServer.Port = adminPort
	cfg.WebServer.User = "nanokvm"
	proxy := proxyConfig{Type: "tcp", LocalIP: "127.0.0.1", LocalPort: localPort, RemotePort: req.RemotePort}
	if previous != nil {
		cfg.Auth.Token = previous.Auth.Token
		cfg.WebServer.Password = previous.WebServer.Password
		proxy.Name = previous.Proxies[0].Name
	} else {
		id, err := randomID()
		if err != nil {
			return nil, err
		}
		proxy.Name = "nanokvm-" + id
		cfg.WebServer.Password, err = randomID()
		if err != nil {
			return nil, err
		}
	}
	if req.Token != nil {
		cfg.Auth.Token = *req.Token
	}
	cfg.Proxies = []proxyConfig{proxy}
	return cfg, nil
}

func readConfig(path string) (*clientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg clientConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.New("configInvalid")
	}
	if err := cfg.validate(); err != nil {
		return nil, errors.New("configInvalid")
	}
	return &cfg, nil
}

func (cfg *clientConfig) validate() error {
	if len(cfg.Proxies) != 1 || cfg.Proxies[0].Name == "" || cfg.Proxies[0].Type != "tcp" ||
		cfg.Proxies[0].LocalIP != "127.0.0.1" || cfg.WebServer.Addr != "127.0.0.1" ||
		cfg.Proxies[0].LocalPort < 1 || cfg.Proxies[0].LocalPort > 65535 ||
		cfg.WebServer.Port != adminPort || cfg.WebServer.Password == "" || cfg.WebServer.User != "nanokvm" ||
		cfg.Auth.Method != "token" || !cfg.Transport.TLS.Enable || !cfg.LoginFailExit {
		return errors.New("configInvalid")
	}
	req := proto.SetFRPConfigReq{ServerAddr: cfg.ServerAddr, ServerPort: cfg.ServerPort, RemotePort: cfg.Proxies[0].RemotePort, Token: &cfg.Auth.Token}
	if err := validateForm(&req); err != nil {
		return err
	}
	cfg.ServerAddr = req.ServerAddr
	return nil
}

func (c *clientConfig) response() proto.FRPConfigRsp {
	return proto.FRPConfigRsp{ServerAddr: c.ServerAddr, ServerPort: c.ServerPort, RemotePort: c.Proxies[0].RemotePort, TokenSet: c.Auth.Token != ""}
}

func (c *clientConfig) accessURL(scheme string) string {
	return scheme + "://" + net.JoinHostPort(c.ServerAddr, strconv.Itoa(c.Proxies[0].RemotePort))
}

// Only the fixed form schema is accepted. Validate in Go and atomically replace
// its serialized TOML without spawning another client on a memory-limited device.
func writeConfig(path string, cfg *clientConfig) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return replaceFile(path, data, 0600)
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".frp-*"+filepath.Ext(path))
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
