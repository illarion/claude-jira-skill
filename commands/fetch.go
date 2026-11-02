package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
	"github.com/illarion/claude-jira-skill/internal/jx"
)

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

type attachment struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mimeType"`
	Size      any    `json:"size"`
	LocalPath any    `json:"local_path"`
}

type comment struct {
	Author  string `json:"author"`
	Created string `json:"created"`
	Body    string `json:"body"`
}

type ticket struct {
	Key         string       `json:"key"`
	Summary     string       `json:"summary"`
	Status      string       `json:"status"`
	Assignee    string       `json:"assignee"`
	Priority    string       `json:"priority"`
	Type        string       `json:"type"`
	Versions    []string     `json:"versions"`
	Description string       `json:"description"`
	Attachments []attachment `json:"attachments"`
	Comments    []comment    `json:"comments"`
}

// Fetch prints a ticket as JSON: fields, comments, and image attachments
// (downloaded, with local_path filled in).
func Fetch(args []string) {
	if len(args) < 1 {
		Fail("Usage: jira fetch ISSUE-KEY")
	}
	key := strings.ToUpper(args[0])
	_, c := start()

	issue := jx.Obj(mustGet(c, "/rest/api/3/issue/"+key+
		"?fields=summary,description,status,assignee,priority,attachment,comment,fixVersions,issuetype"))
	fields := jx.Obj(issue["fields"])

	attachmentsDir := filepath.Join(os.TempDir(), "jira-attachments")
	if err := os.MkdirAll(attachmentsDir, 0o755); err != nil {
		die(err)
	}

	attachments := []attachment{}
	attachmentMap := map[string]string{}
	for _, a := range jx.Arr(fields["attachment"]) {
		att := jx.Obj(a)
		var localPath any
		if strings.HasPrefix(jx.Str(att["mimeType"]), "image/") {
			safe := unsafeName.ReplaceAllString(filepath.Base(jx.Str(att["filename"])), "_")
			p := filepath.Join(attachmentsDir, key+"_"+safe)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				if err := c.Download(jx.Str(att["content"]), p); err != nil {
					p = "DOWNLOAD_FAILED: " + err.Error()
				}
			}
			localPath = p
			attachmentMap[jx.Str(att["filename"])] = p
			if id := jx.Str(att["mediaApiFileId"]); id != "" {
				attachmentMap[id] = p
			}
		}
		attachments = append(attachments, attachment{
			Filename:  jx.Str(att["filename"]),
			MimeType:  jx.Str(att["mimeType"]),
			Size:      att["size"],
			LocalPath: localPath,
		})
	}

	desc := strings.TrimSpace(adf.Extract(fields["description"], attachmentMap))

	commentData := jx.Obj(fields["comment"])
	allComments := jx.Arr(commentData["comments"])
	total := len(allComments)
	if v, ok := commentData["total"].(float64); ok {
		total = int(v)
	}
	for startAt := len(allComments); startAt < total; {
		page := jx.Obj(mustGet(c, fmt.Sprintf("/rest/api/3/issue/%s/comment?startAt=%d&maxResults=50", key, startAt)))
		batch := jx.Arr(page["comments"])
		if len(batch) == 0 {
			break
		}
		allComments = append(allComments, batch...)
		startAt += len(batch)
	}

	comments := []comment{}
	for _, cm := range allComments {
		m := jx.Obj(cm)
		author, ok := jx.Obj(m["author"])["displayName"].(string)
		if !ok {
			author = "Unknown"
		}
		created := cutStr(jx.Str(m["created"]), 16)
		comments = append(comments, comment{
			Author:  author,
			Created: strings.ReplaceAll(created, "T", " "),
			Body:    strings.TrimSpace(adf.Extract(m["body"], attachmentMap)),
		})
	}

	assignee := "Unassigned"
	if a := jx.Obj(fields["assignee"]); a != nil {
		assignee = jx.Str(a["displayName"])
	}
	priority := "None"
	if p := jx.Obj(fields["priority"]); p != nil {
		priority = jx.Str(p["name"])
	}
	versions := []string{}
	for _, v := range jx.Arr(fields["fixVersions"]) {
		versions = append(versions, jx.Str(jx.Obj(v)["name"]))
	}

	printJSON(ticket{
		Key:         key,
		Summary:     jx.Str(fields["summary"]),
		Status:      jx.Str(jx.Obj(fields["status"])["name"]),
		Assignee:    assignee,
		Priority:    priority,
		Type:        jx.Str(jx.Obj(fields["issuetype"])["name"]),
		Versions:    versions,
		Description: desc,
		Attachments: attachments,
		Comments:    comments,
	})
}

func cutStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
