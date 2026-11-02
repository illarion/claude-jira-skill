package digest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/jx"
)

func fromJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return v
}

func TestIsHotfixVersion(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"1.118.2", true},  // patch > 0
		{"1.118.0", false}, // patch == 0
		{"v1.2.3", true},   // non-digits stripped
		{"1.2", false},     // too few parts
		{"release", false}, // no digits
		{"1.2.x", false},   // non-numeric patch
		{"", false},
	}
	for _, c := range cases {
		if got := IsHotfixVersion(c.version); got != c.want {
			t.Errorf("IsHotfixVersion(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

func TestParseChangelogFiltersAndDates(t *testing.T) {
	issue := fromJSON(t, `{"changelog":{"histories":[
		{"created":"2026-01-01T10:00:00.000+0000","author":{"displayName":"Old Dev"},"items":[
			{"field":"status","fromString":"Open","toString":"In Progress"}]},
		{"created":"2026-02-05T10:00:00.000+0000","author":{"displayName":"Alice B"},"items":[
			{"field":"status","fromString":"In Progress","toString":"Review"},
			{"field":"assignee","fromString":"Unassigned","toString":"Alice B"},
			{"field":"priority","fromString":"Low","toString":"High"}]}
	]}}`).(map[string]any)

	all := ParseChangelog(issue, "")
	if len(all) != 3 {
		t.Fatalf("got %d changes, want 3 (status+status+assignee)", len(all))
	}
	if all[0].Author != "Old Dev" || all[1].To != "Review" {
		t.Errorf("unexpected changes: %+v", all)
	}

	feb := ParseChangelog(issue, "2026-02-01")
	if len(feb) != 2 {
		t.Fatalf("got %d changes after cutoff, want 2", len(feb))
	}
	if feb[0].Author != "Alice B" {
		t.Errorf("author = %q", feb[0].Author)
	}
}

func TestDetermineDeveloper(t *testing.T) {
	changes := []Change{
		{Date: "2026-02-03T10:00:00", Author: "Bob", Field: "status", To: "Review"},
		{Date: "2026-02-02T10:00:00", Author: "Alice", Field: "status", To: "In Progress"},
	}
	dev := map[string]bool{"In Progress": true}
	review := map[string]bool{"Review": true}

	// Dev transition wins even though the review one is later in the slice.
	author, _, ok := DetermineDeveloper(changes, dev, review)
	if !ok || author != "Alice" {
		t.Errorf("got %q ok=%v", author, ok)
	}

	// Fallback to review when no dev transition exists.
	author, _, ok = DetermineDeveloper(changes[:1], map[string]bool{}, review)
	if !ok || author != "Bob" {
		t.Errorf("got %q ok=%v", author, ok)
	}

	// Nothing matches.
	if _, _, ok := DetermineDeveloper(changes, map[string]bool{}, map[string]bool{}); ok {
		t.Error("expected no match")
	}
}

func TestDetermineTester(t *testing.T) {
	changes := []Change{
		{Date: "2026-02-02T10:00:00", Author: "Alice", Field: "status", From: "Testing", To: "Tested"},
		{Date: "2026-02-03T10:00:00", Author: "Bob", Field: "status", From: "Open", To: "Tested"},
	}
	testS := map[string]bool{"Testing": true}
	testedS := map[string]bool{"Tested": true}
	author, _, ok := DetermineTester(changes, testS, testedS)
	if !ok || author != "Alice" {
		t.Errorf("got %q ok=%v", author, ok)
	}
	if _, _, ok := DetermineTester(changes[1:], testS, testedS); ok {
		t.Error("from-status must be a test status")
	}
}

func testIssue(t *testing.T) map[string]any {
	return fromJSON(t, `{
		"key": "PROJ-7",
		"fields": {
			"summary": "Fix the thing",
			"status": {"name": "Done"},
			"assignee": {"displayName": "Carol Q."},
			"priority": {"name": "High"},
			"fixVersions": [{"name": "1.118.2"}],
			"description": {"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"It broke"}]}]},
			"customfield_9": {"displayName": "Dave K."}
		},
		"changelog": {
			"total": 2,
			"histories": [
				{"created":"2026-02-02T10:00:00.000+0000","author":{"displayName":"Alice B"},"items":[
					{"field":"status","fromString":"Open","toString":"In Progress"}]},
				{"created":"2026-02-06T10:00:00.000+0000","author":{"displayName":"Eve T."},"items":[
					{"field":"status","fromString":"Testing","toString":"Tested"}]}
			]
		}
	}`).(map[string]any)
}

func TestBuildJSONExecutor(t *testing.T) {
	// Changelog is complete (total == len), so no HTTP fetch happens.
	c := client.New("http://127.0.0.1:1", "e", "t") // would fail loudly if used
	issue := testIssue(t)

	tickets, err := BuildJSON(c, []any{issue}, "2026-02-01", "customfield_9",
		map[string]bool{"In Progress": true}, map[string]bool{}, map[string]bool{"Testing": true}, map[string]bool{"Tested": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 {
		t.Fatal("expected 1 ticket")
	}
	tk := tickets[0]
	if tk.Key != "PROJ-7" || tk.Project != "PROJ" || tk.Status != "Done" {
		t.Errorf("ticket = %+v", tk)
	}
	if tk.Hotfix != true {
		t.Error("1.118.2 should be a hotfix")
	}
	if tk.Developer == nil || *tk.Developer != "Dave K " { // dots replaced by spaces, matching Python
		t.Errorf("developer = %v (executor field should win)", tk.Developer)
	}
	if tk.Tester == nil || *tk.Tester != "Eve T " {
		t.Errorf("tester = %v", tk.Tester)
	}
	if tk.Assignee != "Carol Q " { // dots replaced by spaces, matching Python
		t.Errorf("assignee = %q", tk.Assignee)
	}
	if len(tk.Changelog) != 2 {
		t.Errorf("changelog = %+v", tk.Changelog)
	}
}

func TestBuildJSONDeveloperFromTransition(t *testing.T) {
	// No tester statuses match, so the full changelog is fetched; it comes
	// back empty and the embedded changelog is used, as in the former script.
	mux := httpMux(t, "/rest/api/3/issue/PROJ-7/changelog", `{"values":[],"isLast":true}`)
	defer mux.Close()
	c := client.New(mux.URL, "e", "t")
	issue := testIssue(t)
	delete(jx.Obj(issue["fields"]), "customfield_9")

	tickets, err := BuildJSON(c, []any{issue}, "2026-02-01", "",
		map[string]bool{"In Progress": true}, map[string]bool{}, map[string]bool{}, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	tk := tickets[0]
	if tk.Developer == nil || *tk.Developer != "Alice B" {
		t.Errorf("developer = %v, want Alice B from the In Progress transition", tk.Developer)
	}
	if len(tk.Changelog) != 2 {
		t.Errorf("changelog = %+v, want the embedded entries", tk.Changelog)
	}
}

func TestBuildJSONBeforePeriodNotAttributed(t *testing.T) {
	c := client.New("http://127.0.0.1:1", "e", "t")
	issue := testIssue(t)
	delete(jx.Obj(issue["fields"]), "customfield_9")

	// Reporting period starts after all transitions: nobody is attributed.
	tickets, err := BuildJSON(c, []any{issue}, "2026-03-01", "",
		map[string]bool{"In Progress": true}, map[string]bool{}, map[string]bool{"Testing": true}, map[string]bool{"Tested": true})
	if err != nil {
		t.Fatal(err)
	}
	tk := tickets[0]
	if tk.Developer != nil {
		t.Errorf("developer = %v, want nil", tk.Developer)
	}
	if tk.Tester != nil {
		t.Errorf("tester = %v, want nil", tk.Tester)
	}
	if len(tk.Changelog) != 0 {
		t.Errorf("changelog should be empty for the period, got %+v", tk.Changelog)
	}
}

func TestBuildTeamStats(t *testing.T) {
	alice := "Alice"
	eve := "Eve"
	tickets := []Ticket{
		{Key: "P-0", Hotfix: true}, // hotfix without an attributed developer still counts
		{Key: "P-1", Developer: &alice, Hotfix: true,
			Changelog: []Change{{Date: "2026-02-06", Author: "Bob", Field: "status", From: "Review", To: "Done"}}},
		{Key: "P-2", Developer: &alice},
		{Key: "P-3", Tester: &eve},
	}
	stats := BuildTeamStats(tickets, map[string]bool{"Review": true})

	if stats.TotalTickets != 4 {
		t.Errorf("total_tickets = %v", stats.TotalTickets)
	}
	if stats.TotalHotfixes != 2 {
		t.Errorf("total_hotfixes = %v, want 2 (Python counted every hotfix ticket)", stats.TotalHotfixes)
	}
	if stats.TotalReviews != 1 {
		t.Errorf("total_reviews = %v", stats.TotalReviews)
	}
	a := stats.Person("Alice")
	if a == nil || a.Developed != 2 || a.Hotfixes != 1 || len(a.DevKeys) != 2 {
		t.Errorf("alice = %+v", a)
	}
	if b := stats.Person("Bob"); b == nil || b.Reviews != 1 {
		t.Errorf("bob = %+v", b)
	}
	if e := stats.Person("Eve"); e == nil || e.Tested != 1 || len(e.TestKeys) != 1 {
		t.Errorf("eve = %+v", e)
	}
	// per_person is ordered by developed desc, then first appearance.
	if keys := stats.PerPerson.Keys(); !reflect.DeepEqual(keys, []string{"Alice", "Bob", "Eve"}) {
		t.Errorf("per_person order = %v", keys)
	}
	out, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"total_tickets":4,"total_hotfixes":2,"total_reviews":1,"per_person":{"Alice":`
	if !strings.HasPrefix(string(out), want) {
		t.Errorf("json = %s", out)
	}
}

func TestBuildJSONChangelogFetchErrorIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
	}))
	defer srv.Close()
	c := client.New(srv.URL, "e", "t")
	issue := testIssue(t)
	delete(jx.Obj(issue["fields"]), "customfield_9")

	// No dev status matches, so the full changelog is fetched and fails.
	_, err := BuildJSON(c, []any{issue}, "2026-02-01", "",
		map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{})
	if err == nil || !strings.Contains(err.Error(), "PROJ-7") || !strings.Contains(err.Error(), "Issue does not exist") {
		t.Errorf("err = %v, want the changelog error, not silent empty data", err)
	}
}

func TestBuildReport(t *testing.T) {
	issues := []any{
		fromJSON(t, `{"key":"PROJ-1","fields":{"summary":"Hot thing","status":{"name":"Done"},"assignee":{"displayName":"A"},"fixVersions":[{"name":"1.118.2"}]}}`),
		fromJSON(t, `{"key":"PROJ-2","fields":{"summary":"Normal thing","status":{"name":"In Review"},"assignee":null,"fixVersions":[{"name":"1.119.0"}]}}`),
		fromJSON(t, `{"key":"CORE-3","fields":{"summary":"No version","status":{"name":"Open"},"assignee":null,"fixVersions":[]}}`),
	}
	got := BuildReport(issues)
	want := "### Hotfix Release 1.118.2\n- **Done** — Hot thing (PROJ-1, A)\n\n" +
		"### CORE — Other\n- **Open** — No version (CORE-3, Unassigned)\n\n" +
		"### PROJ — 1.119.0\n- **In Review** — Normal thing (PROJ-2, Unassigned)\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestGetExecutorList(t *testing.T) {
	fields := jx.Obj(fromJSON(t, `{"customfield_1":[{"displayName":"Team A"}],"customfield_2":{"displayName":"Solo Dev"}}`))
	if got := GetExecutor(fields, "customfield_1"); got != "Team A" {
		t.Errorf("list: got %q", got)
	}
	if got := GetExecutor(fields, "customfield_2"); got != "Solo Dev" {
		t.Errorf("single: got %q", got)
	}
	if got := GetExecutor(fields, "customfield_3"); got != "" {
		t.Errorf("missing: got %q", got)
	}
	if got := GetExecutor(fields, ""); got != "" {
		t.Errorf("no field id: got %q", got)
	}
}

func TestFetchProjectStatusesAndFields(t *testing.T) {
	mux := httpMux(t,
		"/rest/api/3/project/PROJ/statuses", `[{"statuses":[{"name":"In Progress"},{"name":"Review"}]},{"statuses":[{"name":"In Progress"}]}]`,
		"/rest/api/3/field", `[{"id":"customfield_1","name":"Zeta","custom":true},{"id":"summary","name":"Summary","custom":false},{"id":"customfield_2","name":"Alpha","custom":true}]`)
	c := client.New(mux.URL, "e", "t")

	names, err := FetchProjectStatuses(c, []string{"PROJ"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"In Progress", "Review"}) {
		t.Errorf("names = %v", names)
	}

	fields, err := FetchCustomFields(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[0].Name != "Alpha" || fields[1].Name != "Zeta" {
		t.Errorf("fields = %+v", fields)
	}
}

// httpMux serves canned JSON responses by path.
func httpMux(t *testing.T, routes ...string) *httptest.Server {
	t.Helper()
	if len(routes)%2 != 0 {
		t.Fatal("routes must be path/body pairs")
	}
	m := map[string]string{}
	for i := 0; i < len(routes); i += 2 {
		m[routes[i]] = routes[i+1]
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := m[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	}))
}
