package types

import "time"

// SyncData represents the saved sync information
type SyncData struct {
	LastKey           string    `json:"last_key"`
	LastSourcePath    string    `json:"last_source_path"`
	LastDestPath      string    `json:"last_dest_path"`
	LastSyncTime      time.Time `json:"last_sync_time"`
	LastOperationMode string    `json:"last_operation_mode"`
}

// UploadThingOpts defines the options used by the entry function.
type UploadThingOpts struct {
	// Mode can be "host" (upload) or "sync" (download)
	Mode string

	// For host mode:
	UpdateExisting bool   // If true, update an existing key.
	Key            string // If empty and not updating, a random 6‐letter key is generated.
	SourcePath     string // Local file or directory to upload.

	// For sync mode:
	DestinationPath string // Local destination path (file or directory).

	// S3/Cloudflare R2 configuration:
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

const (
	SyncBucketRoot = "syncThing/"
)