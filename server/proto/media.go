package proto

import "time"

// Virtual media wire types. A medium is named by a type plus a path relative
// to the media directory: the server resolves the id itself, so a request
// never carries an absolute path it could be talked into deleting.

const (
	MediaTypeImage     = "image"
	MediaTypePartition = "partition"
)

type MediaItem struct {
	Type      string    `json:"type"`
	Rel       string    `json:"rel"`
	Name      string    `json:"name"`
	Size      uint64    `json:"size"`
	FS        string    `json:"fs"`
	Browsable bool      `json:"browsable"`
	Attached  bool      `json:"attached"`
	Modified  time.Time `json:"modifiedAt"`
}

type MediaCaps struct {
	Loop bool `json:"loop"`
}

type GetMediaRsp struct {
	Media []MediaItem `json:"media"`
	Caps  MediaCaps   `json:"capabilities"`
}

type MediaStateRsp struct {
	SharedDiskInitialized bool     `json:"sharedDiskInitialized"` // the shared USB disk image exists
	State                 string   `json:"state"`                 // none | image | partition | unsupported
	Type                  string   `json:"type"`                  // type of the attached medium
	Rel                   string   `json:"rel"`                   // rel of the attached medium, when it is an image
	HidOnly               bool     `json:"hidOnly"`               // the gadget has no mass storage at all
	Open                  []string `json:"open"`                  // media the file manager currently holds open
	Cdrom                 bool     `json:"cdrom"`                 // the attached medium is served as a CD-ROM
	ReadOnly              bool     `json:"readOnly"`
	LocalWritable         bool     `json:"localWritable"` // actual local storage access, after any USB eject recovery
	LocalStorageError     string   `json:"localStorageError,omitempty"`
}

type MediaRefReq struct {
	Challenge string `json:"challenge"`
	Type      string `json:"type" validate:"omitempty,oneof=image partition"`
	Rel       string `json:"rel"`
}

type MountMediaReq struct {
	Type  string `json:"type" validate:"omitempty,oneof=image partition"`
	Rel   string `json:"rel"`
	Cdrom bool   `json:"cdrom"`
	// Omitted by older clients, which always mounted images read-only.
	ReadOnly *bool `json:"readOnly"`
	// Force re-enumerates the gadget to take the medium back from a target
	// machine that is holding it. It costs the target machine its unflushed
	// writes, so the UI asks before setting it.
	Force bool `json:"force"`
}

type EjectMediaReq struct {
	Force bool `json:"force"`
}

type MediaStatusRsp struct {
	Status     string `json:"status"`
	Name       string `json:"name"`
	Percentage string `json:"percentage"`
	Loaded     int64  `json:"loaded"` // bytes transferred so far
	Total      int64  `json:"total"`  // expected bytes, 0 when unknown
	Message    string `json:"message"`
}

type MediaFile struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"isDir"`
	Modified time.Time `json:"modifiedAt"`
}

type GetMediaFilesRsp struct {
	Path    string      `json:"path"`
	Entries []MediaFile `json:"entries"`
	Free    uint64      `json:"free"`
	Total   uint64      `json:"total"`
	// Writing is true while the medium is seeing block level writes: a zero
	// length entry in that state is a transfer in progress, not an empty file.
	Writing bool `json:"writing"`
}

type MediaPathReq struct {
	Challenge string `json:"challenge"`
	Type      string `json:"type" validate:"omitempty,oneof=image partition"`
	Rel       string `json:"rel"`
	Path      string `json:"path" validate:"required"`
}
