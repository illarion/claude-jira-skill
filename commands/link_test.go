package commands

import (
	"testing"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Jira shows the outward phrase ("blocks") on the issue sent as inwardIssue,
// so "PROJ-1 Blocks PROJ-2" must send PROJ-1 as inwardIssue.
func TestLinkBodyDirection(t *testing.T) {
	body := linkBody("Blocks", "PROJ-1", "PROJ-2")

	if got := jx.Str(jx.Obj(body["type"])["name"]); got != "Blocks" {
		t.Errorf("type name = %q", got)
	}
	if got := jx.Str(jx.Obj(body["inwardIssue"])["key"]); got != "PROJ-1" {
		t.Errorf("inwardIssue = %q, want PROJ-1", got)
	}
	if got := jx.Str(jx.Obj(body["outwardIssue"])["key"]); got != "PROJ-2" {
		t.Errorf("outwardIssue = %q, want PROJ-2", got)
	}
}
