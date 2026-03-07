package attach

import "time"

// FileType represents the classified type of an attachment.
type FileType string

const (
	FileTypeText       FileType = "text"
	FileTypeLog        FileType = "log"
	FileTypeStructured FileType = "structured"
	FileTypeImage      FileType = "image"
	FileTypeCode       FileType = "code"
	FileTypeBinary     FileType = "binary"
)

// Attachment represents a file dragged/pasted into VectorPad.
// Content is never stored — only the path reference.
type Attachment struct {
	Path     string    // absolute path on disk
	Name     string    // basename
	Type     FileType  // classified type
	Label    string    // display label: [text], [log], [json], etc.
	Size     int64     // bytes
	Lines    int       // line count for text types, -1 for binary/image
	Modified time.Time // last modification time
}

// SerializeMode controls how an attachment appears in copy-out.
type SerializeMode string

const (
	SerializePathOnly SerializeMode = "path"     // just the path
	SerializeExcerpt  SerializeMode = "excerpt"  // metadata + excerpt (default)
	SerializeEvidence SerializeMode = "evidence" // operator-curated block
)
