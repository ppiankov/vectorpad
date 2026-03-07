package attach

import "os"

// ImageProtocol represents a terminal image rendering protocol.
type ImageProtocol string

const (
	ImageProtocolNone   ImageProtocol = "none"
	ImageProtocolITerm2 ImageProtocol = "iterm2"
	ImageProtocolKitty  ImageProtocol = "kitty"
)

// DetectImageProtocol checks terminal environment for image support.
// Called once at startup, result cached for the session.
func DetectImageProtocol() ImageProtocol {
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return ImageProtocolITerm2
	}
	if os.Getenv("TERM") == "xterm-kitty" {
		return ImageProtocolKitty
	}
	return ImageProtocolNone
}

// SupportsImages returns true if the terminal can render inline images.
func SupportsImages(proto ImageProtocol) bool {
	return proto != ImageProtocolNone
}
