package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/storage/diskfs"
	"NanoKVM-Server/utils"
)

type fileUploadPlan struct {
	id        string
	mediaType string
	rel       string
	target    string
	size      int64
	chunks    int64
}

type mediaFileUpload struct {
	mu            sync.Mutex
	plan          fileUploadPlan
	medium        diskfs.Image
	file          *diskfs.Upload
	releaseTarget func()
	done          chan struct{}
	next          int64
	written       int64
	closed        bool
	generation    uint64
	timer         *time.Timer
}

func parseFileUpload(c *gin.Context) (fileUploadPlan, int64, error) {
	invalid := func() (fileUploadPlan, int64, error) {
		return fileUploadPlan{}, 0, fmt.Errorf("%w: invalid file upload metadata", utils.ErrInvalidMediaPath)
	}
	plan := fileUploadPlan{id: c.Query("uploadId"), mediaType: c.Query("type"), rel: c.Query("rel")}
	if !validImageUploadID(plan.id) || (plan.mediaType != proto.MediaTypePartition && plan.mediaType != proto.MediaTypeImage) {
		return invalid()
	}
	dir, err := utils.ValidateMediumPath(c.Query("path"))
	if err != nil {
		return invalid()
	}
	name := c.Query("filename")
	if err := utils.ValidateFileName(name); err != nil || diskfs.IsTransferTemp(name) {
		return invalid()
	}
	plan.target = name
	if dir != "/" {
		plan.target = path.Join(dir, name)
	}
	plan.size, err = strconv.ParseInt(c.Query("size"), 10, 64)
	if err != nil || plan.size < 0 || plan.size > uploadMaxBytes {
		return invalid()
	}
	chunkSize, err := strconv.ParseInt(c.Query("chunkSize"), 10, 64)
	if err != nil || chunkSize != imageUploadChunkSizeBytes {
		return invalid()
	}
	plan.chunks, err = strconv.ParseInt(c.Query("totalChunks"), 10, 64)
	wanted := max(int64(1), (plan.size+chunkSize-1)/chunkSize)
	if err != nil || plan.chunks != wanted {
		return invalid()
	}
	index, err := strconv.ParseInt(c.Query("chunkIndex"), 10, 64)
	if err != nil || index < 0 || index >= plan.chunks {
		return invalid()
	}
	if plan.mediaType == proto.MediaTypePartition {
		plan.rel = ""
	}
	return plan, index, nil
}

// The returned session is locked. Its job slot and medium handle span every
// request, so USB transitions and deleting its destination remain blocked
// between chunks. Other uploads can queue without buffering their bodies.
func (s *Service) lockFileUpload(ctx context.Context, plan fileUploadPlan, index int64) (*mediaFileUpload, error) {
	s.fileUploadMu.Lock()
	upload := s.fileUploads[plan.id]
	if upload != nil {
		s.fileUploadMu.Unlock()
		upload.mu.Lock()
		if upload.closed || upload.plan != plan {
			upload.mu.Unlock()
			return nil, fmt.Errorf("%w: upload session changed", utils.ErrInvalidMediaPath)
		}
		return upload, nil
	}
	if index != 0 {
		s.fileUploadMu.Unlock()
		return nil, fmt.Errorf("%w: upload must start with the first chunk", utils.ErrInvalidMediaPath)
	}
	if len(s.fileUploads) >= maxQueuedJobs+1 {
		s.fileUploadMu.Unlock()
		return nil, errTooManyJobs
	}
	if s.fileUploads == nil {
		s.fileUploads = make(map[string]*mediaFileUpload)
	}
	upload = &mediaFileUpload{plan: plan}
	upload.mu.Lock()
	s.fileUploads[plan.id] = upload
	s.fileUploadMu.Unlock()

	var err error
	upload.done, err = s.beginJob(ctx, plan.target)
	if err == nil {
		upload.medium, err = openMedium(plan.mediaType, plan.rel, true)
	}
	if err == nil {
		var free uint64
		_, free, err = upload.medium.StatFS()
		if err == nil && uint64(plan.size) > free {
			err = errNoSpace
		}
	}
	if err == nil {
		upload.releaseTarget, err = reserveMediaUpload(plan.mediaType, plan.target)
	}
	if err == nil {
		upload.file, err = upload.medium.BeginUpload(plan.target)
	}
	if err != nil {
		s.closeFileUpload(upload, jobStateFailed, err.Error())
		upload.mu.Unlock()
		return nil, err
	}
	s.setJobProgress(upload.done, 0, plan.size)
	return upload, nil
}

func (s *Service) closeFileUpload(upload *mediaFileUpload, state jobState, message string) {
	if upload.closed {
		return
	}
	upload.closed = true
	if upload.timer != nil {
		upload.timer.Stop()
	}
	if upload.file != nil {
		closeMedia(upload.file)
	}
	if upload.releaseTarget != nil {
		upload.releaseTarget()
	}
	if upload.medium != nil {
		closeMedia(upload.medium)
	}
	if upload.done != nil {
		s.finishJob(upload.done, state, message)
	}
	s.fileUploadMu.Lock()
	if s.fileUploads[upload.plan.id] == upload {
		delete(s.fileUploads, upload.plan.id)
	}
	s.fileUploadMu.Unlock()
}

func (s *Service) expireFileUpload(upload *mediaFileUpload, generation uint64) {
	upload.mu.Lock()
	defer upload.mu.Unlock()
	if !upload.closed && upload.generation == generation {
		s.closeFileUpload(upload, jobStateFailed, "file upload expired")
	}
}

func (s *Service) uploadMediaFileChunk(c *gin.Context) {
	plan, index, err := parseFileUpload(c)
	if err != nil {
		respondError(c, err)
		return
	}
	size := min(imageUploadChunkSizeBytes, plan.size-index*imageUploadChunkSizeBytes)
	if c.Request.ContentLength > size || (c.Request.ContentLength >= 0 && c.Request.ContentLength != size) {
		respondError(c, fmt.Errorf("%w: invalid chunk length", utils.ErrInvalidMediaPath))
		return
	}
	upload, err := s.lockFileUpload(c.Request.Context(), plan, index)
	if err != nil {
		respondError(c, err)
		return
	}
	defer upload.mu.Unlock()
	if index != upload.next {
		respondError(c, fmt.Errorf("%w: upload chunk out of order", utils.ErrInvalidMediaPath))
		return
	}
	upload.generation++
	if upload.timer != nil {
		upload.timer.Stop()
	}

	// Share the image uploader's single chunk slot: a file and an ISO upload
	// must not allocate two chunk working sets at the same time.
	release, err := defaultImageUploadDependencies.limiter.acquire(c.Request.Context())
	if err == nil {
		defer release()
		var free uint64
		_, free, err = upload.medium.StatFS()
		if err == nil && uint64(plan.size-upload.written) > free {
			err = errNoSpace
		}
	}
	if err == nil {
		body := http.MaxBytesReader(c.Writer, c.Request.Body, size)
		offset := upload.written
		progress := utils.NewTransferProgressWriter(io.Discard, size, func(loaded, _ int64) {
			s.setJobProgress(upload.done, offset+loaded, plan.size)
		})
		var written int64
		written, err = upload.file.WriteChunk(io.TeeReader(body, progress), size)
		progress.Stop()
		if err == nil {
			upload.written += written
			upload.next++
		}
	}
	if err == nil {
		err = c.Request.Context().Err()
	}
	complete := upload.next == plan.chunks
	if err == nil && complete {
		err = upload.file.Commit()
	}
	if err != nil {
		s.closeFileUpload(upload, jobStateFailed, err.Error())
		respondError(c, err)
		return
	}
	s.setJobProgress(upload.done, upload.written, plan.size)
	if complete {
		s.closeFileUpload(upload, jobStateSuccess, "")
	} else {
		generation := upload.generation
		upload.timer = time.AfterFunc(uploadStorageIdleTimeout, func() { s.expireFileUpload(upload, generation) })
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, gin.H{"uploadId": plan.id, "written": upload.written, "complete": complete})
}

func (s *Service) CancelMediaFileUpload(c *gin.Context) {
	var req struct {
		UploadID string `json:"uploadId"`
	}
	if err := proto.ParseFormRequest(c, &req); err != nil || !validImageUploadID(req.UploadID) {
		respondError(c, utils.ErrInvalidMediaPath)
		return
	}
	s.fileUploadMu.Lock()
	upload := s.fileUploads[req.UploadID]
	s.fileUploadMu.Unlock()
	if upload != nil {
		upload.mu.Lock()
		s.closeFileUpload(upload, jobStateIdle, "")
		upload.mu.Unlock()
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}
