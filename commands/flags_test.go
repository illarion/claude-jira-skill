package commands

import (
	"flag"
	"testing"
)

func TestParseFlagsAfterPositionals(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	priority := fs.String("priority", "", "p")
	jsonFlag := fs.Bool("json", false, "j")
	max := fs.Int("max", 0, "m")

	pos := parseFlags(fs, []string{"PROJ-1", "--priority", "Major", "--json", "--max", "20"})

	if len(pos) != 1 || pos[0] != "PROJ-1" {
		t.Errorf("positionals = %v", pos)
	}
	if *priority != "Major" || !*jsonFlag || *max != 20 {
		t.Errorf("flags: priority=%q json=%v max=%d", *priority, *jsonFlag, *max)
	}
}

func TestParseFlagsMixedAndEquals(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	a := fs.String("a", "", "a")
	b := fs.String("b", "", "b")
	c := fs.String("c", "", "c")

	pos := parseFlags(fs, []string{"--a", "1", "pos1", "--b=2", "pos2", "--c", "3"})

	if len(pos) != 2 || pos[0] != "pos1" || pos[1] != "pos2" {
		t.Errorf("positionals = %v", pos)
	}
	if *a != "1" || *b != "2" || *c != "3" {
		t.Errorf("flags: a=%q b=%q c=%q", *a, *b, *c)
	}
}

func TestParseFlagsDoubleDashStopsFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	a := fs.String("a", "", "a")
	pos := parseFlags(fs, []string{"--a", "x", "--", "-not-a-flag", "pos"})
	if *a != "x" {
		t.Errorf("a = %q", *a)
	}
	if len(pos) != 2 || pos[0] != "-not-a-flag" {
		t.Errorf("positionals = %v", pos)
	}
}

func TestParseFlagsBoolDoesNotConsumeNext(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	list := fs.Bool("list", false, "l")
	pos := parseFlags(fs, []string{"--list", "KEY"})
	if !*list {
		t.Error("--list not set")
	}
	if len(pos) != 1 || pos[0] != "KEY" {
		t.Errorf("positionals = %v", pos)
	}
}

func TestParseFlagsDashAndNegativeValues(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	desc := fs.String("description", "", "d")
	max := fs.Int("max", 0, "m")
	pos := parseFlags(fs, []string{"KEY", "--description", "-", "--max", "-1"})
	if *desc != "-" || *max != -1 {
		t.Errorf("description=%q max=%d", *desc, *max)
	}
	if len(pos) != 1 || pos[0] != "KEY" {
		t.Errorf("positionals = %v", pos)
	}
}
