package attach

import "testing"

func TestDetectImageProtocolDefault(t *testing.T) {
	// In test environment, neither iTerm2 nor kitty env vars are set.
	proto := DetectImageProtocol()
	if proto != ImageProtocolNone && proto != ImageProtocolITerm2 && proto != ImageProtocolKitty {
		t.Errorf("unexpected protocol: %s", proto)
	}
}

func TestSupportsImages(t *testing.T) {
	if SupportsImages(ImageProtocolNone) {
		t.Error("none should not support images")
	}
	if !SupportsImages(ImageProtocolITerm2) {
		t.Error("iterm2 should support images")
	}
	if !SupportsImages(ImageProtocolKitty) {
		t.Error("kitty should support images")
	}
}
