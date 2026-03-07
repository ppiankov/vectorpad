package attach

import (
	"fmt"
	"strings"
)

// Serialize converts an attachment to text-safe output for copy-out.
// No binary data ever enters the copy payload.
func Serialize(a *Attachment, mode SerializeMode, excerptLines int) string {
	if a == nil {
		return ""
	}

	switch mode {
	case SerializePathOnly:
		return fmt.Sprintf("[Attached: %s]", a.Path)

	case SerializeEvidence:
		return serializeEvidence(a, excerptLines)

	default: // SerializeExcerpt
		return serializeExcerpt(a, excerptLines)
	}
}

func serializeExcerpt(a *Attachment, maxLines int) string {
	var b strings.Builder

	meta := FormatSize(a.Size)
	if a.Lines >= 0 {
		meta += fmt.Sprintf(", %d lines", a.Lines)
	}

	switch a.Type {
	case FileTypeImage:
		fmt.Fprintf(&b, "Attached image: %s (%s)\n", a.Name, meta)
	default:
		fmt.Fprintf(&b, "Attached %s: %s (%s)\n", a.Type, a.Name, meta)
	}

	if IsTextType(a.Type) {
		preview := Preview(a, maxLines)
		if preview != "" {
			b.WriteString("Relevant excerpt:\n")
			for _, line := range strings.Split(preview, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func serializeEvidence(a *Attachment, maxLines int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Evidence from %s:\n", a.Name)

	if IsTextType(a.Type) {
		preview := Preview(a, maxLines)
		if preview != "" {
			for _, line := range strings.Split(preview, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	} else {
		fmt.Fprintf(&b, "  [%s, %s]\n", a.Type, FormatSize(a.Size))
	}

	return strings.TrimRight(b.String(), "\n")
}
