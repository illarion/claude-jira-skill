// Package client talks to the Jira REST API with Basic auth,
// rendering API errors like the former Python scripts did.
package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Client issues authenticated requests against one Jira instance.
type Client struct {
	BaseURL string
	Email   string
	Token   string
	HTTP    *http.Client
}

// New builds a client with a 30s timeout.
func New(baseURL, email, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Email:   email,
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.Email+":"+c.Token))
}

// Get, Post, Put are conveniences around Request.
func (c *Client) Get(path string) (any, error) { return c.Request("GET", path, nil) }
func (c *Client) Post(path string, data any) (any, error) {
	return c.Request("POST", path, data)
}
func (c *Client) Put(path string, data any) (any, error) {
	return c.Request("PUT", path, data)
}

// Request performs one API call. data is marshaled as the JSON body when
// non-nil. It returns the parsed response, or nil for an empty body.
func (c *Client) Request(method, path string, data any) (any, error) {
	var body []byte
	if data != nil {
		var err error
		if body, err = json.Marshal(data); err != nil {
			return nil, err
		}
	}
	_, raw, err := c.do(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON from %s: %w", path, err)
	}
	return out, nil
}

// Download fetches a full URL (an attachment) with a 60s timeout and writes
// it to dest. Redirects drop the Authorization header automatically, which
// is what Jira's signed attachment URLs expect.
func (c *Client) Download(fullURL, dest string) error {
	dl := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	resp, err := dl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(dest)
		return err
	}
	return f.Close()
}

const maxAttempts = 5

// do performs the request, retrying rate limits (429) with Retry-After or
// exponential backoff, and turning HTTP >= 400 into readable errors.
func (c *Client) do(method, fullURL string, body []byte) (*http.Response, []byte, error) {
	for attempt := 0; ; attempt++ {
		var rdr io.Reader
		if body != nil {
			rdr = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, fullURL, rdr)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", c.authHeader())
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, nil, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 429 && attempt < maxAttempts-1 {
			delay := time.Duration(1<<attempt) * time.Second
			if ra, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && ra > int(delay.Seconds()) {
				delay = time.Duration(ra) * time.Second
			}
			fmt.Fprintf(os.Stderr, "Rate limited, retrying in %ds...\n", int(delay.Seconds()))
			time.Sleep(delay)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, nil, httpError(resp.StatusCode, raw)
		}
		return resp, raw, nil
	}
}

// httpError mirrors the Python error rendering: Jira's errorMessages plus
// per-field errors, or a plain "HTTP <code>: <reason>" fallback.
func httpError(code int, raw []byte) error {
	var body struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(raw, &body); err == nil && (len(body.ErrorMessages) > 0 || len(body.Errors) > 0) {
		parts := append([]string{}, body.ErrorMessages...)
		keys := make([]string, 0, len(body.Errors))
		for k := range body.Errors {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, k+": "+body.Errors[k])
		}
		return fmt.Errorf("%s", strings.Join(parts, "\n"))
	}
	if json.Valid(raw) && len(raw) > 0 {
		return fmt.Errorf("HTTP %d: %s", code, raw)
	}
	return fmt.Errorf("HTTP %d: %s", code, http.StatusText(code))
}

// Search runs a JQL search over Jira v3's cursor-paginated endpoint.
// max <= 0 means unbounded (all pages). expand is passed through when set.
func Search(c *Client, jql, fields string, max int, expand string) ([]any, error) {
	issues := []any{}
	nextToken := ""
	for {
		pageSize := 50
		if max > 0 {
			pageSize = max - len(issues)
			if pageSize > 50 {
				pageSize = 50
			}
			if pageSize <= 0 {
				break
			}
		}
		q := url.Values{}
		q.Set("jql", jql)
		q.Set("maxResults", strconv.Itoa(pageSize))
		q.Set("fields", fields)
		if expand != "" {
			q.Set("expand", expand)
		}
		if nextToken != "" {
			q.Set("nextPageToken", nextToken)
		}
		data, err := c.Get("/rest/api/3/search/jql?" + q.Encode())
		if err != nil {
			return nil, err
		}
		m := jx.Obj(data)
		batch := jx.Arr(m["issues"])
		issues = append(issues, batch...)
		if max > 0 && len(issues) >= max {
			break
		}
		if len(batch) == 0 || jx.Bool(m["isLast"], true) {
			break
		}
		nextToken = jx.Str(m["nextPageToken"])
		if nextToken == "" {
			break
		}
	}
	if max > 0 && len(issues) > max {
		issues = issues[:max]
	}
	return issues, nil
}
