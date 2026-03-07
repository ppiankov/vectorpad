package attach

// ExcerptRange defines which lines to include in the copy-out.
type ExcerptRange struct {
	Start int // 0-indexed start line
	End   int // 0-indexed end line (exclusive), 0 = use default
}

// ExcerptConfig holds per-attachment serialization settings.
type ExcerptConfig struct {
	Mode  SerializeMode
	Range ExcerptRange
	Lines int // override default line count, 0 = use default
}

// DefaultExcerptConfig returns the default config for an attachment.
func DefaultExcerptConfig(a *Attachment) ExcerptConfig {
	if a == nil {
		return ExcerptConfig{Mode: SerializeExcerpt}
	}

	lines := defaultHeadLines
	if a.Type == FileTypeLog {
		lines = defaultTailLines
	}

	return ExcerptConfig{
		Mode:  SerializeExcerpt,
		Lines: lines,
	}
}
