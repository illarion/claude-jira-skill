// Package adf converts between light markup, ADF documents and plain text.
//
// Jira Cloud only accepts ADF; wiki syntax renders as literal text. The
// converters here mirror the former Python implementation exactly, including
// the supported markup: #/# headings, - bullets, 1. ordered items, ``` code
// fences, **bold**, `code`, bare URLs and issue keys (linked when jiraURL is
// given).
package adf

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

var (
	inlineRe  = regexp.MustCompile(`(\*\*[^*]+\*\*|` + "`[^`]+`" + `|https?://[^\s<>()]+|\b[A-Z][A-Z0-9]+-\d+\b)`)
	headingRe = regexp.MustCompile(`^(#{1,6}) (.+)$`)
	bulletRe  = regexp.MustCompile(`^[-*] (.+)$`)
	orderedRe = regexp.MustCompile(`^(\d+)\. (.+)$`)
)

// TextToADF converts light markup into an ADF document.
func TextToADF(text, jiraURL string) map[string]any {
	doc := []any{}
	var curList map[string]any
	var code []string
	codeLang := ""
	codeOpen := false

	flushCode := func() {
		doc = append(doc, codeBlock(code, codeLang))
		code, codeLang, codeOpen = nil, "", false
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)

		if codeOpen {
			if strings.HasPrefix(line, "```") {
				flushCode()
			} else {
				code = append(code, raw)
			}
			continue
		}
		if strings.HasPrefix(line, "```") {
			code = []string{}
			codeOpen = true
			codeLang = strings.TrimSpace(line[3:])
			curList = nil
			continue
		}
		if line == "" {
			curList = nil
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			doc = append(doc, map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": len(m[1])},
				"content": parseInline(m[2], jiraURL),
			})
			curList = nil
			continue
		}
		if m := bulletRe.FindStringSubmatch(line); m != nil {
			curList = ensureList(&doc, curList, "bulletList", m[1], jiraURL, 0)
			continue
		}
		if m := orderedRe.FindStringSubmatch(line); m != nil {
			order, _ := strconv.Atoi(m[1])
			curList = ensureList(&doc, curList, "orderedList", m[2], jiraURL, order)
			continue
		}
		doc = append(doc, paragraph(line, jiraURL))
		curList = nil
	}
	if codeOpen {
		flushCode()
	}
	return map[string]any{"version": 1, "type": "doc", "content": doc}
}

// LoadDescription reads a description file: a raw ADF document when the file
// is JSON with type "doc", light markup otherwise.
func LoadDescription(path, jiraURL string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err == nil {
		if t, _ := doc["type"].(string); t == "doc" {
			return doc, nil
		}
	}
	return TextToADF(string(b), jiraURL), nil
}

// Extract renders an ADF node back to readable text (used by fetch).
// attachmentMap maps attachment filenames / media ids to local download paths.
func Extract(node any, attachmentMap map[string]string) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	t := jx.Str(m["type"])
	switch t {
	case "text":
		text := jx.Str(m["text"])
		for _, mk := range jx.Arr(m["marks"]) {
			switch jx.Str(jx.Obj(mk)["type"]) {
			case "code":
				text = "`" + text + "`"
			case "strong":
				text = "**" + text + "**"
			}
		}
		return text
	case "hardBreak":
		return "\n"
	case "mediaSingle", "mediaGroup":
		parts := []string{}
		for _, child := range jx.Arr(m["content"]) {
			parts = append(parts, Extract(child, attachmentMap))
		}
		return strings.Join(parts, "\n")
	case "media":
		attrs := jx.Obj(m["attrs"])
		alt := jx.Str(attrs["alt"])
		mediaID := jx.Str(attrs["id"])
		if attachmentMap != nil {
			if p, ok := attachmentMap[alt]; ok && alt != "" {
				return "\n[IMAGE: " + alt + " -> " + p + "]\n"
			}
			if p, ok := attachmentMap[mediaID]; ok && mediaID != "" {
				return "\n[IMAGE: " + p + "]\n"
			}
		}
		name := alt
		if name == "" {
			name = mediaID
		}
		return "\n[IMAGE: " + name + "]\n"
	case "codeBlock":
		lang := jx.Str(jx.Obj(m["attrs"])["language"])
		lines := []string{}
		for _, child := range jx.Arr(m["content"]) {
			lines = append(lines, Extract(child, attachmentMap))
		}
		return "\n```" + lang + "\n" + strings.Join(lines, "") + "\n```\n"
	case "heading":
		level := jx.Int(jx.Obj(m["attrs"])["level"])
		if level == 0 {
			level = 1
		}
		parts := []string{}
		for _, child := range jx.Arr(m["content"]) {
			parts = append(parts, Extract(child, attachmentMap))
		}
		return "\n" + strings.Repeat("#", level) + " " + strings.Join(parts, "") + "\n"
	case "bulletList", "orderedList":
		items := []string{}
		for i, child := range jx.Arr(m["content"]) {
			prefix := "- "
			if t == "orderedList" {
				prefix = strconv.Itoa(i+1) + ". "
			}
			items = append(items, prefix+strings.TrimSpace(Extract(child, attachmentMap)))
		}
		return strings.Join(items, "\n") + "\n"
	case "listItem":
		parts := []string{}
		for _, child := range jx.Arr(m["content"]) {
			if s := strings.TrimSpace(Extract(child, attachmentMap)); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	case "inlineCard":
		return jx.Str(jx.Obj(m["attrs"])["url"])
	default:
		parts := []string{}
		for _, child := range jx.Arr(m["content"]) {
			parts = append(parts, Extract(child, attachmentMap))
		}
		result := strings.Join(parts, "")
		if t == "paragraph" {
			result += "\n"
		}
		return result
	}
}

// ExtractText flattens an ADF node to plain text without markup,
// words joined by spaces (used by the digest descriptions).
func ExtractText(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	parts := []string{}
	if jx.Str(m["type"]) == "text" {
		parts = append(parts, jx.Str(m["text"]))
	}
	for _, child := range jx.Arr(m["content"]) {
		parts = append(parts, ExtractText(child))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func parseInline(text, jiraURL string) []any {
	nodes := []any{}
	last := 0
	flushText := func(end int) {
		if end > last {
			nodes = append(nodes, map[string]any{"type": "text", "text": text[last:end]})
		}
	}
	for _, loc := range inlineRe.FindAllStringSubmatchIndex(text, -1) {
		token := text[loc[0]:loc[1]]
		if isIssueKey(token) && !wordBounded(text, loc[0], loc[1]) {
			continue
		}
		flushText(loc[0])
		nodes = append(nodes, inlineNode(token, jiraURL))
		last = loc[1]
	}
	flushText(len(text))
	return nodes
}

func isIssueKey(token string) bool {
	return !strings.HasPrefix(token, "**") && !strings.HasPrefix(token, "`") && !strings.HasPrefix(token, "http")
}

// wordBounded reports whether text[start:end] sits on Unicode word
// boundaries. RE2's \b is ASCII-only; Python's is Unicode-aware, so an issue
// key glued to a Cyrillic word must stay plain text.
func wordBounded(text string, start, end int) bool {
	before, _ := utf8.DecodeLastRuneInString(text[:start])
	after, _ := utf8.DecodeRuneInString(text[end:])
	return !isWordRune(before) && !isWordRune(after)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func inlineNode(token, jiraURL string) map[string]any {
	switch {
	case strings.HasPrefix(token, "**"):
		return map[string]any{"type": "text", "text": token[2 : len(token)-2], "marks": []any{map[string]any{"type": "strong"}}}
	case strings.HasPrefix(token, "`"):
		return map[string]any{"type": "text", "text": token[1 : len(token)-1], "marks": []any{map[string]any{"type": "code"}}}
	case strings.HasPrefix(token, "http"):
		return map[string]any{"type": "text", "text": token, "marks": []any{map[string]any{"type": "link", "attrs": map[string]any{"href": token}}}}
	case jiraURL == "":
		return map[string]any{"type": "text", "text": token}
	default:
		return map[string]any{"type": "text", "text": token, "marks": []any{map[string]any{"type": "link", "attrs": map[string]any{"href": jiraURL + "/browse/" + token}}}}
	}
}

func paragraph(text, jiraURL string) map[string]any {
	return map[string]any{"type": "paragraph", "content": parseInline(text, jiraURL)}
}

func codeBlock(lines []string, lang string) map[string]any {
	attrs := map[string]any{}
	if lang != "" {
		attrs = map[string]any{"language": lang}
	}
	content := []any{}
	if len(lines) > 0 {
		content = append(content, map[string]any{"type": "text", "text": strings.Join(lines, "\n")})
	}
	return map[string]any{"type": "codeBlock", "attrs": attrs, "content": content}
}

func ensureList(doc *[]any, cur map[string]any, listType, text, jiraURL string, order int) map[string]any {
	if cur == nil || cur["type"] != listType {
		l := map[string]any{"type": listType, "content": []any{}}
		if listType == "orderedList" {
			l["attrs"] = map[string]any{"order": order}
		}
		*doc = append(*doc, l)
		cur = l
	}
	item := map[string]any{"type": "listItem", "content": []any{paragraph(text, jiraURL)}}
	cur["content"] = append(cur["content"].([]any), item)
	return cur
}
