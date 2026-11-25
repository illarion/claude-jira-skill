package commands

import (
	"strings"
	"testing"
	"time"
)

// fetch must download an attachment of any type; only the size can skip it.
func TestSkipReason(t *testing.T) {
	if got := skipReason(maxAttachmentBytes, false); got != "" {
		t.Errorf("at the limit: got %q, want download", got)
	}
	if got := skipReason(maxAttachmentBytes+1, true); got != "" {
		t.Errorf("over the limit with --all-attachments: got %q, want download", got)
	}
	got := skipReason(maxAttachmentBytes+1, false)
	if !strings.Contains(got, "25 MB") || !strings.Contains(got, "--all-attachments") {
		t.Errorf("over the limit: got %q", got)
	}
}

func TestDownloadTimeout(t *testing.T) {
	if got := downloadTimeout(maxAttachmentBytes); got != 60*time.Second {
		t.Errorf("at the limit: got %v", got)
	}
	if got := downloadTimeout(maxAttachmentBytes + 1); got != 30*time.Minute {
		t.Errorf("over the limit: got %v", got)
	}
}

// Two attachments of one ticket can have the same file name. Each must get
// its own local path, or the second is taken for the first and not downloaded.
func TestAttachmentPathSameFilename(t *testing.T) {
	first := attachmentPath("/tmp/att", "PROJ-1", "10001", "image.png")
	second := attachmentPath("/tmp/att", "PROJ-1", "10002", "image.png")
	if first == second {
		t.Errorf("both attachments got %q", first)
	}
	if want := "/tmp/att/PROJ-1_10001_image.png"; first != want {
		t.Errorf("got %q want %q", first, want)
	}
	if got := attachmentPath("/tmp/att", "PROJ-1", "../1", "a/b c.png"); got != "/tmp/att/PROJ-1_.._1_b_c.png" {
		t.Errorf("unsafe id or name: got %q", got)
	}
}
