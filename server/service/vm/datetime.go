package vm

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	setSystemTimeRequestMaxBytes = 2048
	localDatetimeLayout          = "2006-01-02T15:04:05"
)

var (
	getSystemTimeLocation     = loadSystemTimeLocation
	prepareSystemTimeSessions = authservice.PrepareSystemTimeSessionInvalidation
	setSystemTime             = setLinuxSystemTime
)

var systemTimeMutationMu sync.Mutex

var (
	minimumManualSystemTime = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	maximumManualSystemTime = time.Unix(1<<31-1, 0)
)

func (s *Service) SetTimeZone(c *gin.Context) {
	var req proto.SetTimeZoneReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if _, err := time.LoadLocation(req.Timezone); err != nil {
		rsp.ErrRsp(c, -2, "invalid timezone")
		return
	}

	systemTimeMutationMu.Lock()
	defer systemTimeMutationMu.Unlock()

	cmd := exec.Command("timedatectl", "set-timezone", req.Timezone)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("set timezone failed, output: %s", string(output))
		rsp.ErrRsp(c, -2, "set timezone failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set timezone %s success", req.Timezone)
}

func (s *Service) GetTimeZone(c *gin.Context) {
	var rsp proto.Response

	timezone, err := getIANATimezone()
	if err != nil {
		rsp.ErrRsp(c, -1, "failed to get timezone")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTimeZoneRsp{
		Timezone: timezone,
	})
	log.Debugf("get timezone %s success", timezone)
}

func getIANATimezone() (string, error) {
	tzPath, err := os.Readlink("/etc/localtime")
	if err != nil {
		log.Errorf("Failed to read timezone from /etc/localtime: %s", err)
		return "", err
	}

	zoneinfo := "/usr/share/zoneinfo/"
	if i := strings.LastIndex(tzPath, zoneinfo); i != -1 {
		return tzPath[i+len(zoneinfo):], nil
	}

	if !filepath.IsAbs(tzPath) {
		tzPath = filepath.Join("/etc", tzPath)
	}

	relPath, err := filepath.Rel(zoneinfo, tzPath)
	if err != nil {
		log.Errorf("Failed to find timezone from %s: %s", tzPath, err)
		return "", err
	}

	return relPath, nil
}

func (s *Service) GetTime(c *gin.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	ctx := c.Request.Context()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case t := <-ticker.C:
			c.SSEvent("message", t.UnixMilli())
			return true
		}
	})
}

func (s *Service) SetTime(c *gin.Context) {
	var rsp proto.Response
	var req proto.SetTimeReq

	if err := decodeSetTimeJSON(c, &req); err != nil {
		log.Debugf("invalid manual time request: %s", err)
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if req.Datetime == "" {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	systemTimeMutationMu.Lock()
	defer systemTimeMutationMu.Unlock()

	location, err := getSystemTimeLocation()
	if err != nil {
		log.Errorf("get timezone for manual time setting failed: %s", err)
		rsp.ErrRsp(c, -2, "failed to get timezone")
		return
	}
	requestedTime, err := time.ParseInLocation(localDatetimeLayout, req.Datetime, location)
	if err != nil || requestedTime.Format(localDatetimeLayout) != req.Datetime ||
		requestedTime.Before(minimumManualSystemTime) || requestedTime.After(maximumManualSystemTime) ||
		isAmbiguousLocalTime(requestedTime, location) {
		rsp.ErrRsp(c, -1, "invalid datetime")
		return
	}

	activateSessionInvalidation, err := prepareSystemTimeSessions()
	if err != nil {
		log.Errorf("prepare session invalidation before setting system time failed: %s", err)
		rsp.ErrRsp(c, -2, "prepare session invalidation failed")
		return
	}

	if err := setSystemTime(requestedTime); err != nil {
		activateSessionInvalidation(false)
		log.Errorf("set system time failed: %s", err)
		rsp.ErrRsp(c, -2, "set system time failed")
		return
	}
	activateSessionInvalidation(true)

	rsp.OkRspWithData(c, &proto.SetTimeRsp{Timestamp: requestedTime.UnixMilli()})
	log.Infof("system time set manually in timezone %s; authentication sessions invalidated", location)
}

func decodeSetTimeJSON(c *gin.Context, destination interface{}) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return errors.New("missing request body")
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return errors.New("content type must be application/json")
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, setSystemTimeRequestMaxBytes)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func loadSystemTimeLocation() (*time.Location, error) {
	timezone, err := getIANATimezone()
	if err != nil {
		return nil, err
	}
	return time.LoadLocation(timezone)
}

func setLinuxSystemTime(value time.Time) error {
	timespec, err := unix.TimeToTimespec(value)
	if err != nil {
		return err
	}
	return unix.ClockSettime(unix.CLOCK_REALTIME, &timespec)
}

func isAmbiguousLocalTime(value time.Time, location *time.Location) bool {
	if location == nil {
		return false
	}

	year, month, day := value.Date()
	hour, minute, second := value.Clock()
	wallTimeUTC := time.Date(year, month, day, hour, minute, second, 0, time.UTC)

	// Offsets on both sides of a clock transition identify every instant that
	// can represent this wall-clock value. The widest modern IANA transitions
	// are well within this two-day window.
	offsets := make(map[int]struct{})
	for _, delta := range []time.Duration{-48 * time.Hour, -24 * time.Hour, 0, 24 * time.Hour, 48 * time.Hour} {
		_, offset := value.Add(delta).In(location).Zone()
		offsets[offset] = struct{}{}
	}

	instants := make(map[int64]struct{})
	for offset := range offsets {
		candidate := wallTimeUTC.Add(-time.Duration(offset) * time.Second).In(location)
		candidateYear, candidateMonth, candidateDay := candidate.Date()
		candidateHour, candidateMinute, candidateSecond := candidate.Clock()
		if candidateYear == year && candidateMonth == month && candidateDay == day &&
			candidateHour == hour && candidateMinute == minute && candidateSecond == second {
			instants[candidate.Unix()] = struct{}{}
		}
	}

	return len(instants) > 1
}

func (s *Service) GetTimeStatus(c *gin.Context) {
	var rsp proto.Response

	status, err := utils.GetChronySyncStatus()
	if err != nil {
		log.Errorf("get chrony sync status failed: %s", err)
		rsp.ErrRsp(c, -1, "get chrony sync status failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTimeStatusRsp{
		IsSynchronized: status.IsSynchronized,
		LastSyncTime:   status.LastSyncTime,
	})
	log.Debugf("get chrony sync status success")
}

func (s *Service) SyncTime(c *gin.Context) {
	systemTimeMutationMu.Lock()
	defer systemTimeMutationMu.Unlock()

	var rsp proto.Response

	err := utils.ChronycImmediateSync()
	if err != nil {
		log.Errorf("chrony immediate sync failed: %s", err)
		rsp.ErrRsp(c, -1, "sync time failed")
		return
	}

	time.Sleep(2 * time.Second)

	rsp.OkRsp(c)
	log.Debugf("sync time success")
}
