package storage

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/service/storage/diskfs"
	"NanoKVM-Server/storageguard"
	"NanoKVM-Server/utils"
)

const (
	codeInvalidArgument   = -1
	codeOperationFailed   = -2
	codeStorageDisabled   = -10
	codeMediumLocked      = -11
	codeMediumBusy        = -12
	codeNotBrowsable      = -13
	codeInsufficientSpace = -14
	codeNotOptical        = -15
	partitionDevice       = "/exfat.img"
)

var (
	mediaDirectory        = imageDirectory
	sharedDiskInitialized = func() bool { return pathExists(partitionDevice) }
	mediumOperationMu     sync.Mutex
	mediaWriteTargets     = make(map[string]bool)
	errStorageDisabled    = errors.New("initialize the virtual disk in Settings > Device first")
	errPartitionOwned     = errors.New("eject the shared folder before writing local storage")
	errMediumAttached     = errors.New("eject the image before accessing its contents")
	errMediumBusy         = errors.New("the medium is in use by a file transfer or the file manager")
	errNoSpace            = errors.New("not enough space on the medium")
	errNotOptical         = errors.New("a CD-ROM can only carry an ISO 9660 or UDF image")
)

func (s *Service) GetMedia(c *gin.Context) {
	if err := refreshSharedStorage(); err != nil {
		respondError(c, err)
		return
	}
	release, err := acquireStorageAccess(false)
	if err != nil {
		respondError(c, err)
		return
	}
	defer release()
	media, err := listMedia()
	if err != nil {
		respondError(c, err)
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &proto.GetMediaRsp{Media: media, Caps: proto.MediaCaps{Loop: diskfs.Probe().Loop}})
}

func (s *Service) GetMediaState(c *gin.Context) {
	// Opening/polling storage also returns ownership after a host-side eject.
	// Keep a successful local lease until the USB snapshot is collected.
	release, accessErr := acquireStorageAccess(true)
	if accessErr == nil {
		defer release()
	}
	status, err := readDiskStatus()
	if err != nil {
		respondError(c, err)
		return
	}
	data := &proto.MediaStateRsp{SharedDiskInitialized: sharedDiskInitialized(), State: string(status.State), HidOnly: status.HidOnly, Open: openMediaIDs(), Cdrom: status.Cdrom, ReadOnly: status.ReadOnly || status.Cdrom}
	data.LocalWritable = accessErr == nil
	if accessErr != nil && !errors.Is(accessErr, storageguard.ErrBusy) {
		if !errors.Is(accessErr, storageguard.ErrReadOnly) {
			data.LocalStorageError = accessErr.Error()
		} else if status.State != diskStatePartition {
			data.LocalStorageError = "local storage is read-only"
		}
	}
	if status.State == diskStateImage {
		data.Type = proto.MediaTypeImage
		data.Rel = mediaRel(status.Backing)
	}
	if status.State == diskStatePartition {
		data.Type = proto.MediaTypePartition
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, data)
}

func (s *Service) GetMediaStatus(c *gin.Context) {
	s.jobMutex.Lock()
	data := &proto.MediaStatusRsp{Status: string(s.jobState), Name: s.jobName, Percentage: utils.FormatPercentage(s.jobLoaded, s.jobTotal), Loaded: s.jobLoaded, Total: s.jobTotal, Message: s.jobMessage}
	s.jobMutex.Unlock()
	var rsp proto.Response
	rsp.OkRspWithData(c, data)
}

func (s *Service) MountMedia(c *gin.Context) {
	var req proto.MountMediaReq
	if err := proto.ParseFormRequest(c, &req); err != nil {
		respondError(c, utils.ErrInvalidMediaID)
		return
	}
	backing := partitionDevice
	if req.Type != proto.MediaTypePartition {
		// Resolve again after handing /data back from USB: that transition
		// replaces the old loop mount and its cached directory entries.
		if req.Rel == "" || !validImageName(filepath.Join(mediaDirectory, req.Rel), mediaDirectory) || !filepath.IsLocal(req.Rel) {
			respondError(c, utils.ErrInvalidMediaID)
			return
		}
		backing = filepath.Join(mediaDirectory, req.Rel)
	}
	readOnly := req.ReadOnly == nil || *req.ReadOnly
	if err := s.attachMedium(backing, req.Cdrom, readOnly, req.Force); err != nil {
		respondError(c, err)
		return
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}

func (s *Service) EjectMedia(c *gin.Context) {
	var req proto.EjectMediaReq
	if c.Request.ContentLength != 0 {
		if err := proto.ParseFormRequest(c, &req); err != nil {
			respondError(c, utils.ErrInvalidMediaID)
			return
		}
	}
	if err := s.detachMedium(req.Force); err != nil {
		respondError(c, err)
		return
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}

func (s *Service) DeleteMedia(c *gin.Context) {
	var req proto.MediaRefReq
	if err := proto.ParseFormRequest(c, &req); err != nil || req.Type == proto.MediaTypePartition {
		respondError(c, utils.ErrInvalidMediaID)
		return
	}
	if !verifyMediaDelete(c, req.Challenge) {
		return
	}
	release, err := acquireStorageAccess(true)
	if err != nil {
		respondError(c, err)
		return
	}
	defer release()
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	backing, err := utils.ResolveMediaID(req.Rel)
	if err == nil {
		err = guardMediaDelete(backing)
	}
	if err == nil {
		err = os.Remove(backing)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}

func verifyMediaDelete(c *gin.Context, challenge string) bool {
	if deleteImageActionRequiresChallenge() {
		if err := verifyDeleteImageActionChallenge(c, challenge, authservice.SensitiveActionDeleteImage); err != nil {
			var rsp proto.Response
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return false
		}
	}
	return true
}

func guardMediaDelete(target string) error {
	for path := range mediaWriteTargets {
		if pathWithin(path, target) {
			return errMediumBusy
		}
	}
	status, err := readDiskStatus()
	if err != nil {
		return err
	}
	if status.State == diskStatePartition {
		return errPartitionOwned
	}
	return deleteGuard(target, status.Backing, diskfs.BusyImages())
}

// Reserve the unpublished destination too: another transfer must not
// replace it or remove its parent before this upload's atomic rename.
func reserveMediaUpload(mediaType, target string) (func(), error) {
	if mediaType != proto.MediaTypePartition {
		return func() {}, nil
	}
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	full, err := utils.CanonicalPathInside(mediaDirectory, target)
	if err != nil {
		return nil, err
	}
	if err := guardMediaDelete(full); err != nil {
		return nil, err
	}
	mediaWriteTargets[full] = true
	return func() { mediumOperationMu.Lock(); delete(mediaWriteTargets, full); mediumOperationMu.Unlock() }, nil
}

func (s *Service) DownloadMedia(c *gin.Context) {
	if c.Query("type") == proto.MediaTypePartition {
		respondError(c, utils.ErrInvalidMediaID)
		return
	}
	if err := refreshSharedStorage(); err != nil {
		respondError(c, err)
		return
	}
	release, err := acquireStorageAccess(false)
	if err != nil {
		respondError(c, err)
		return
	}
	defer release()
	mediumOperationMu.Lock()
	backing, err := utils.ResolveMediaID(c.Query("rel"))
	var releasePin func()
	if err == nil {
		releasePin, err = pinImageDownload(backing)
	}
	mediumOperationMu.Unlock()
	if err != nil {
		respondError(c, err)
		return
	}
	defer releasePin()
	file, err := os.Open(backing)
	if err != nil {
		respondError(c, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		respondError(c, errors.New("not a regular image file"))
		return
	}
	c.Header("Content-Disposition", attachmentHeader(info.Name()))
	c.Header("Content-Type", "application/octet-stream")
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}

// The caller holds mediumOperationMu so checking the gadget and pinning the
// image cannot race a local writer, deletion, or image replacement.
func pinImageDownload(backing string) (func(), error) {
	status, err := readDiskStatus()
	if err != nil {
		return nil, err
	}
	if status.State == diskStateImage && sameMediaPath(status.Backing, backing) {
		return nil, errMediumAttached
	}
	return diskfs.AcquireReadPin(backing)
}

func listMedia() ([]proto.MediaItem, error) {
	status, err := readDiskStatus()
	if err != nil {
		return nil, err
	}
	if err := ensureImageDirectory(mediaDirectory); err != nil {
		return nil, err
	}
	media := []proto.MediaItem{}
	err = filepath.Walk(mediaDirectory, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.Mode().IsRegular() || !utils.IsMediaFile(info.Name()) || diskfs.IsTransferTemp(info.Name()) {
			return nil
		}
		kind, _ := diskfs.Detect(path)
		media = append(media, proto.MediaItem{Type: proto.MediaTypeImage, Rel: mediaRel(path), Name: info.Name(), Size: uint64(info.Size()), FS: string(kind), Browsable: kind.Browsable(), Attached: status.State == diskStateImage && sameMediaPath(status.Backing, path), Modified: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(media, func(i, j int) bool { return media[i].Rel < media[j].Rel })
	if sharedDiskInitialized() {
		media = append(media, proto.MediaItem{Type: proto.MediaTypePartition, Rel: "", Name: "Shared folder", FS: "exfat", Browsable: true, Attached: status.State == diskStatePartition})
	}
	return media, nil
}

func mediaRel(backing string) string { return strings.TrimPrefix(backing, mediaDirectory+"/") }

func openMediaIDs() []string {
	paths := diskfs.BusyImages()
	ids := make([]string, 0, len(paths))
	for _, path := range paths {
		ids = append(ids, mediaRel(path))
	}
	return ids
}

func openMedium(mediaType, rel string, writable bool) (diskfs.Image, error) {
	if !writable {
		if err := refreshSharedStorage(); err != nil {
			return nil, err
		}
	}
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	return openMediumLocked(mediaType, rel, writable)
}

// Every handle retains the cross-process storage lock, including readers.
// USB rebuilds and a fresh /data mount must wait for the stream to close.
func openMediumLocked(mediaType, rel string, writable bool) (diskfs.Image, error) {
	if mediaType != "" && mediaType != proto.MediaTypeImage && mediaType != proto.MediaTypePartition {
		return nil, utils.ErrInvalidMediaID
	}
	release, err := acquireStorageAccess(writable)
	if err != nil {
		return nil, err
	}
	// Local image storage may use the ordinary /data directory before USB disk
	// initialization. File sharing must only expose the initialized shared disk.
	if mediaType == proto.MediaTypePartition && !sharedDiskInitialized() {
		release()
		return nil, errStorageDisabled
	}
	var image diskfs.Image
	status, err := readDiskStatus()
	if err == nil && writable && status.State == diskStatePartition {
		err = errPartitionOwned
	}
	if err == nil && mediaType == proto.MediaTypePartition {
		image, err = diskfs.OpenDirectory(mediaDirectory)
	} else if err == nil {
		var backing string
		backing, err = utils.ResolveMediaID(rel)
		if err == nil && status.State == diskStatePartition {
			err = errPartitionOwned
		}
		if err == nil && status.State == diskStateImage && sameMediaPath(status.Backing, backing) {
			err = errMediumAttached
		}
		if err == nil {
			image, err = diskfs.Mount(backing, writable)
		}
	}
	if err != nil {
		release()
		return nil, err
	}
	return &storageMediaHandle{Image: image, release: release}, nil
}

type storageMediaHandle struct {
	diskfs.Image
	release func()
	once    sync.Once
	err     error
}

func (h *storageMediaHandle) Close() error {
	h.once.Do(func() { h.err = h.Image.Close(); h.release() })
	return h.err
}

func sameMediaPath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

func deleteGuard(target, backing string, mounted []string) error {
	if backing != "" && pathWithin(backing, target) {
		return errMediumAttached
	}
	for _, path := range mounted {
		if pathWithin(path, target) {
			return errMediumBusy
		}
	}
	return nil
}

func pathWithin(path, dir string) bool {
	path = strings.ToUpper(filepath.Clean(path))
	dir = strings.ToUpper(filepath.Clean(dir))
	return path == dir || strings.HasPrefix(path, dir+"/")
}

func respondError(c *gin.Context, err error) {
	var rsp proto.Response
	code := codeOperationFailed
	switch {
	case errors.Is(err, utils.ErrInvalidMediaID), errors.Is(err, utils.ErrInvalidMediaPath):
		code = codeInvalidArgument
	case errors.Is(err, errStorageDisabled):
		code = codeStorageDisabled
	case errors.Is(err, ErrMediumLocked):
		code = codeMediumLocked
	case errors.Is(err, errMediumBusy), errors.Is(err, errMediumAttached), errors.Is(err, errPartitionOwned), errors.Is(err, diskfs.ErrBusy), errors.Is(err, storageguard.ErrBusy), errors.Is(err, storageguard.ErrReadOnly):
		code = codeMediumBusy
	case errors.Is(err, diskfs.ErrUnsupported):
		code = codeNotBrowsable
	case errors.Is(err, errNoSpace), errors.Is(err, errTooManyJobs):
		code = codeInsufficientSpace
	case errors.Is(err, errNotOptical):
		code = codeNotOptical
	}
	rsp.ErrRsp(c, code, err.Error())
}

func attachmentHeader(name string) string {
	fallback := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, rfc5987Encode(name))
}

// rfc5987Encode percent-encodes every byte that is not an RFC 5987 attr-char.
// url.PathEscape is not that set: it leaves "=", ":", "@" and friends alone,
// which a strict parser reads as the end of the value.
func rfc5987Encode(value string) string {
	const hex = "0123456789ABCDEF"

	var encoded strings.Builder
	for i := 0; i < len(value); i++ {
		b := value[i]
		if isAttrChar(b) {
			encoded.WriteByte(b)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hex[b>>4])
		encoded.WriteByte(hex[b&0x0f])
	}
	return encoded.String()
}

func isAttrChar(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9':
		return true
	}
	return strings.IndexByte("!#$&+-.^_`|~", b) >= 0
}

// Keep the legacy download endpoint's idle-on-completion contract. The new
// page also needs the final result, so it follows this dedicated endpoint.
func (s *Service) GetMediaTransferStatus(c *gin.Context) {
	s.downloadMu.Lock()
	running := s.download != nil
	result := s.lastDownload
	s.downloadMu.Unlock()
	terminal := result.Status == "success" || result.Status == "failed" || result.Status == "checksum_failed"
	if !terminal && (running || pathExists(sentinelPath)) {
		s.StatusImage(c)
		return
	}
	if result.Status == "" {
		result = newImageDownloadStatus("idle", "", "")
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &result)
}
