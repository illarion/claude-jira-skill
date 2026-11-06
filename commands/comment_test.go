package commands

import (
	"strings"
	"testing"
)

func TestCommentArgsPositionalText(t *testing.T) {
	key, text, file, err := commentArgs([]string{"proj-1", "Fixed in latest commit"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if key != "PROJ-1" || text != "Fixed in latest commit" || file != "" {
		t.Errorf("key=%q text=%q file=%q", key, text, file)
	}
}

func TestCommentArgsDescriptionFile(t *testing.T) {
	key, text, file, err := commentArgs([]string{"COREB-2666", "--description-file", "/tmp/desc.md"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if key != "COREB-2666" || text != "" || file != "/tmp/desc.md" {
		t.Errorf("key=%q text=%q file=%q", key, text, file)
	}
}

func TestCommentArgsRejectsFlagLikeText(t *testing.T) {
	_, _, _, err := commentArgs([]string{"PROJ-1", "--", "--description-file"})
	if err == nil || !strings.Contains(err.Error(), "looks like a flag") {
		t.Errorf("err = %v", err)
	}
}

func TestCommentArgsRejectsMissingBody(t *testing.T) {
	for _, args := range [][]string{{}, {"PROJ-1"}} {
		if _, _, _, err := commentArgs(args); err == nil {
			t.Errorf("args %v: expected error", args)
		}
	}
}

func TestCommentArgsRejectsBothForms(t *testing.T) {
	_, _, _, err := commentArgs([]string{"PROJ-1", "text", "--description-file", "/tmp/desc.md"})
	if err == nil || !strings.Contains(err.Error(), "not both") {
		t.Errorf("err = %v", err)
	}
}
