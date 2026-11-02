package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

func TestRequestErrorWithMessagesAndFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages":["Bad thing"],"errors":{"summary":"is required"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	_, err := c.Get("/rest/api/3/x")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "Bad thing\nsummary: is required"
	if err.Error() != want {
		t.Errorf("got %q want %q", err.Error(), want)
	}
}

func TestRequestErrorPlainFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`issue not found`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	_, err := c.Get("/x")
	if err == nil || err.Error() != "HTTP 404: Not Found" {
		t.Errorf("got %v", err)
	}
}

func TestRequestErrorJSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"x":1}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	_, err := c.Get("/x")
	if err == nil || err.Error() != "HTTP 403: {\"x\":1}" {
		t.Errorf("got %v", err)
	}
}

func TestRetryOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	res, err := c.Get("/x")
	if err != nil {
		t.Fatal(err)
	}
	if !jx.Bool(jx.Obj(res)["ok"], false) {
		t.Errorf("res = %v", res)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d", calls.Load())
	}
}

func TestEmptyBodyReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	res, err := c.Request("DELETE", "/x", nil)
	if err != nil || res != nil {
		t.Errorf("res=%v err=%v", res, err)
	}
}

func TestSearchPagination(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("jql") == "" {
			t.Errorf("missing jql")
		}
		calls.Add(1)
		if r.URL.Query().Get("nextPageToken") == "" {
			w.Write([]byte(`{"issues":[{"key":"A-1"}],"isLast":false,"nextPageToken":"t2"}`))
			return
		}
		w.Write([]byte(`{"issues":[{"key":"A-2"},{"key":"A-3"}],"isLast":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	issues, err := Search(c, "project = A", "summary", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 3 {
		t.Fatalf("got %d issues", len(issues))
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d", calls.Load())
	}
}

func TestSearchMaxCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("maxResults") != "2" {
			t.Errorf("maxResults = %s", r.URL.Query().Get("maxResults"))
		}
		w.Write([]byte(`{"issues":[{"key":"A-1"},{"key":"A-2"},{"key":"A-3"}],"isLast":false,"nextPageToken":"t2"}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	issues, err := Search(c, "project = A", "summary", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Errorf("got %d issues, want the --max cap of 2", len(issues))
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing auth header")
		}
		w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()
	c := New(srv.URL, "e", "t")
	dest := t.TempDir() + "/att.png"
	if err := c.Download(srv.URL+"/file", dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "PNGDATA" {
		t.Errorf("data=%q err=%v", data, err)
	}
}
