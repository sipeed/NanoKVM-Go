package vm

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
)

const ScriptDirectory = "/etc/kvm/scripts"

func (s *Service) GetScripts(c *gin.Context) {
	var rsp proto.Response

	var files []string
	err := filepath.Walk(ScriptDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isScript(info.Name()) {
			files = append(files, info.Name())
		}

		return nil
	})
	if err != nil {
		rsp.ErrRsp(c, -1, "get scripts failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetScriptsRsp{
		Files: files,
	})

	log.Debugf("get scripts total %d", len(files))
}

func (s *Service) UploadScript(c *gin.Context) {
	var rsp proto.Response

	if authservice.ScriptUploadRequiresChallenge() {
		if err := authservice.VerifySensitiveActionChallenge(c, c.PostForm("challenge"), authservice.SensitiveActionUploadScript); err != nil {
			log.Errorf("verify script upload challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		rsp.ErrRsp(c, -1, "bad request")
		return
	}
	defer file.Close()

	name, err := validateScriptName(header.Filename)
	if err != nil {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if err = os.MkdirAll(ScriptDirectory, 0o755); err != nil {
		rsp.ErrRsp(c, -2, "create directory failed")
		return
	}

	target, err := safeScriptPath(name)
	if err != nil {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		rsp.ErrRsp(c, -2, "create file failed")
		return
	}
	defer dst.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(dst, file, buf)
	if err != nil {
		rsp.ErrRsp(c, -2, "save failed")
		return
	}

	_ = os.Chmod(target, 0o644)

	data := &proto.UploadScriptRsp{
		File: name,
	}
	rsp.OkRspWithData(c, data)

	log.Debugf("upload script %s success", name)
}

func (s *Service) RunScript(c *gin.Context) {
	var req proto.RunScriptReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if authservice.ScriptExecutionRequiresChallenge() {
		if err := authservice.VerifySensitiveActionChallenge(c, req.Challenge, authservice.SensitiveActionRunScript); err != nil {
			log.Errorf("verify script run challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	scriptPath, err := resolveScriptPath(req.Name)
	if err != nil {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	info, err := os.Stat(scriptPath)
	if err != nil || info.IsDir() {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	var output []byte
	cmd, err := scriptCommand(req.Name, scriptPath)
	if err != nil {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if req.Type == "foreground" {
		output, err = cmd.CombinedOutput()
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
		go func() {
			err := cmd.Run()
			if err != nil {
				log.Errorf("run script %s in background failed: %s", req.Name, err)
			}
		}()
	}

	if err != nil {
		log.Errorf("run script %s failed: %s", req.Name, err.Error())
		rsp.ErrRsp(c, -2, "run script failed")
		return
	}

	rsp.OkRspWithData(c, &proto.RunScriptRsp{
		Log: string(output),
	})

	log.Debugf("run script %s success", req.Name)
}

func (s *Service) DeleteScript(c *gin.Context) {
	var req proto.DeleteScriptReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if authservice.ScriptDeleteRequiresChallenge() {
		if err := authservice.VerifySensitiveActionChallenge(c, req.Challenge, authservice.SensitiveActionDeleteScript); err != nil {
			log.Errorf("verify script delete challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	file, err := safeScriptPath(req.Name)
	if err != nil {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}
	if info, err := os.Stat(file); err != nil || info.IsDir() {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if err := os.Remove(file); err != nil {
		log.Errorf("delete script %s failed: %s", file, err)
		rsp.ErrRsp(c, -3, "delete failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("delete script %s success", file)
}

func isScript(name string) bool {
	nameLower := strings.ToLower(name)
	if strings.HasSuffix(nameLower, ".sh") || strings.HasSuffix(nameLower, ".py") {
		return true
	}

	return false
}

func validateScriptName(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return "", errors.New("invalid script name")
	}
	if name == ".sh" || name == ".py" {
		return "", errors.New("invalid script name")
	}

	for _, char := range name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '.' || char == '_' || char == '-' {
			continue
		}

		return "", errors.New("invalid script name")
	}

	if !isScript(name) {
		return "", errors.New("invalid script type")
	}

	return name, nil
}

func safeScriptPath(name string) (string, error) {
	name, err := validateScriptName(name)
	if err != nil {
		return "", err
	}

	cleanDirectory, err := filepath.Abs(filepath.Clean(ScriptDirectory))
	if err != nil {
		return "", err
	}

	scriptPath, err := filepath.Abs(filepath.Join(cleanDirectory, name))
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(cleanDirectory, scriptPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", errors.New("invalid script path")
	}

	return scriptPath, nil
}

func resolveScriptPath(name string) (string, error) {
	scriptPath, err := safeScriptPath(name)
	if err != nil {
		return "", err
	}

	cleanDirectory := filepath.Clean(ScriptDirectory)
	cleanPath := filepath.Clean(scriptPath)
	if filepath.Dir(cleanPath) != cleanDirectory {
		return "", errors.New("invalid script path")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("script path is directory")
	}

	return cleanPath, nil
}

func scriptCommand(name, scriptPath string) (*exec.Cmd, error) {
	nameLower := strings.ToLower(name)
	var cmd *exec.Cmd
	if strings.HasSuffix(nameLower, ".py") {
		cmd = exec.Command("python3", scriptPath)
	} else if strings.HasSuffix(nameLower, ".sh") {
		cmd = exec.Command("/bin/sh", scriptPath)
	} else {
		return nil, errors.New("invalid script type")
	}

	cmd.Dir = ScriptDirectory
	cmd.Env = []string{
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
	}
	return cmd, nil
}
