package adf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// fromJSON parses a JSON string into generic any values for comparison.
func fromJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return v
}

func checkTextToADF(t *testing.T, text, jiraURL, want string) {
	t.Helper()
	got := TextToADF(text, jiraURL)
	if !reflect.DeepEqual(fromJSON(t, want), fromJSON(t, toJSON(got))) {
		t.Errorf("TextToADF(%q) mismatch:\n got: %s\nwant: %s", text, toJSON(got), want)
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestTextToADFParagraph(t *testing.T) {
	checkTextToADF(t, "hello world", "", `{
		"version": 1, "type": "doc",
		"content": [{"type": "paragraph", "content": [{"type": "text", "text": "hello world"}]}]
	}`)
}

func TestTextToADFInline(t *testing.T) {
	checkTextToADF(t, "see **bold** and `code` and PROJ-12 here", "https://j.example",
		`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[
			{"type":"text","text":"see "},
			{"type":"text","text":"bold","marks":[{"type":"strong"}]},
			{"type":"text","text":" and "},
			{"type":"text","text":"code","marks":[{"type":"code"}]},
			{"type":"text","text":" and "},
			{"type":"text","text":"PROJ-12","marks":[{"type":"link","attrs":{"href":"https://j.example/browse/PROJ-12"}}]},
			{"type":"text","text":" here"}
		]}]}`)
}

func TestTextToADFURLWithoutInstance(t *testing.T) {
	// Without a Jira URL the issue key stays plain text.
	got := TextToADF("PROJ-12", "")
	if !reflect.DeepEqual(fromJSON(t, `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"PROJ-12"}]}]}`),
		fromJSON(t, toJSON(got))) {
		t.Errorf("got %s", toJSON(got))
	}
}

func TestTextToADFBareURL(t *testing.T) {
	checkTextToADF(t, "go to https://example.com/x now", "",
		`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[
			{"type":"text","text":"go to "},
			{"type":"text","text":"https://example.com/x","marks":[{"type":"link","attrs":{"href":"https://example.com/x"}}]},
			{"type":"text","text":" now"}
		]}]}`)
}

func TestTextToADFHeading(t *testing.T) {
	checkTextToADF(t, "### Steps", "", `{
		"version":1,"type":"doc",
		"content":[{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"Steps"}]}]
	}`)
}

func TestTextToADFLists(t *testing.T) {
	checkTextToADF(t, "- a\n- b\n1. first\n2. second\n3. third", "", `{
		"version":1,"type":"doc","content":[
			{"type":"bulletList","content":[
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
			]},
			{"type":"orderedList","attrs":{"order":1},"content":[
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]},
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]},
				{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"third"}]}]}
			]}
		]}`)
}

func TestTextToADFListBrokenByParagraph(t *testing.T) {
	// A paragraph between bullets must split the list into two.
	got := TextToADF("- a\nmiddle\n- b", "")
	doc := got["content"].([]any)
	if len(doc) != 3 {
		t.Fatalf("expected 3 blocks, got %d: %s", len(doc), toJSON(got))
	}
	if doc[0].(map[string]any)["type"] != "bulletList" ||
		doc[1].(map[string]any)["type"] != "paragraph" ||
		doc[2].(map[string]any)["type"] != "bulletList" {
		t.Errorf("wrong block sequence: %s", toJSON(got))
	}
}

func TestTextToADFCodeFence(t *testing.T) {
	checkTextToADF(t, "```go\nfmt.Println(1)\nfmt.Println(2)\n```", "", `{
		"version":1,"type":"doc",
		"content":[{"type":"codeBlock","attrs":{"language":"go"},
			"content":[{"type":"text","text":"fmt.Println(1)\nfmt.Println(2)"}]}]}`)
}

func TestTextToADFUnclosedFence(t *testing.T) {
	checkTextToADF(t, "```\nabc", "", `{
		"version":1,"type":"doc",
		"content":[{"type":"codeBlock","attrs":{},"content":[{"type":"text","text":"abc"}]}]}`)
}

func TestTextToADFCodeInsideFenceNotParsed(t *testing.T) {
	// Inline markup inside a fence must stay literal.
	got := TextToADF("```\n**not bold**\n```", "")
	block := got["content"].([]any)[0].(map[string]any)
	text := block["content"].([]any)[0].(map[string]any)["text"]
	if text != "**not bold**" {
		t.Errorf("fence content changed: %v", text)
	}
}

func TestLoadDescriptionRawADF(t *testing.T) {
	raw := `{"version":1,"type":"doc","content":[{"type":"panel","attrs":{"panelType":"info"}}]}`
	path := filepath.Join(t.TempDir(), "desc.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDescription(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromJSON(t, raw), fromJSON(t, toJSON(got))) {
		t.Errorf("raw ADF not passed through: %s", toJSON(got))
	}
}

func TestLoadDescriptionMarkup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desc.md")
	if err := os.WriteFile(path, []byte("- step one"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDescription(path, "")
	if err != nil {
		t.Fatal(err)
	}
	doc := got["content"].([]any)
	if len(doc) != 1 || doc[0].(map[string]any)["type"] != "bulletList" {
		t.Errorf("markup not converted: %s", toJSON(got))
	}
}

func TestLoadDescriptionJSONButNotDoc(t *testing.T) {
	// Valid JSON that is not an ADF document is treated as markup.
	path := filepath.Join(t.TempDir(), "desc.txt")
	if err := os.WriteFile(path, []byte(`[1, 2]`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDescription(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if got["type"] != "doc" {
		t.Errorf("expected fallback to markup conversion, got %s", toJSON(got))
	}
}

func TestExtractMarks(t *testing.T) {
	node := fromJSON(t, `{"type":"paragraph","content":[
		{"type":"text","text":"a "},
		{"type":"text","text":"b","marks":[{"type":"strong"}]},
		{"type":"text","text":"c","marks":[{"type":"code"}]},
		{"type":"hardBreak"},
		{"type":"text","text":"d"}
	]}`)
	want := "a **b**`c`\nd\n"
	if got := Extract(node, nil); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestExtractHeadingAndLists(t *testing.T) {
	node := fromJSON(t, `{"type":"doc","content":[
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Title"}]},
		{"type":"bulletList","content":[
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}
		]},
		{"type":"orderedList","content":[
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]}
		]}
	]}`)
	want := "\n## Title\n- one\n- two\n1. first\n"
	if got := Extract(node, nil); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestExtractCodeBlock(t *testing.T) {
	node := fromJSON(t, `{"type":"codeBlock","attrs":{"language":"js"},"content":[{"type":"text","text":"console.log(1);"}]}`)
	want := "\n```js\nconsole.log(1);\n```\n"
	if got := Extract(node, nil); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestExtractMediaWithAttachmentMap(t *testing.T) {
	node := fromJSON(t, `{"type":"media","attrs":{"alt":"screenshot.png","id":"m1"}}`)
	byAlt := map[string]string{"screenshot.png": "/tmp/jira-attachments/PROJ-1_screenshot.png"}
	want := "\n[IMAGE: screenshot.png -> /tmp/jira-attachments/PROJ-1_screenshot.png]\n"
	if got := Extract(node, byAlt); got != want {
		t.Errorf("got %q want %q", got, want)
	}
	byID := map[string]string{"m1": "/tmp/x.png"}
	want2 := "\n[IMAGE: /tmp/x.png]\n"
	if got := Extract(node, byID); got != want2 {
		t.Errorf("got %q want %q", got, want2)
	}
	want3 := "\n[IMAGE: screenshot.png]\n"
	if got := Extract(node, nil); got != want3 {
		t.Errorf("got %q want %q", got, want3)
	}
}

func TestExtractText(t *testing.T) {
	node := fromJSON(t, `{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]},
		{"type":"paragraph","content":[{"type":"text","text":"bye"}]}
	]}`)
	if got := ExtractText(node); got != "hello world bye" {
		t.Errorf("got %q", got)
	}
	if got := ExtractText(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestTextToADFIssueKeyNeedsUnicodeWordBoundary(t *testing.T) {
	// Python's \b is Unicode-aware: a key glued to Cyrillic letters is not a key.
	checkTextToADF(t, "тикетPROJ-1 и PROJ-2я, но PROJ-3", "https://j.example",
		`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[
			{"type":"text","text":"тикетPROJ-1 и PROJ-2я, но "},
			{"type":"text","text":"PROJ-3","marks":[{"type":"link","attrs":{"href":"https://j.example/browse/PROJ-3"}}]}
		]}]}`)
}
