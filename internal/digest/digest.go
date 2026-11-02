// Package digest computes the period team-digest report: who developed,
// tested and reviewed each ticket, derived from the issue changelog.
package digest

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Change is one status/assignee transition from a changelog.
type Change struct {
	Date   string `json:"date"`
	Author string `json:"author"`
	Field  string `json:"field"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// IsHotfixVersion reports whether a fix version looks like a hotfix:
// at least three numeric parts and a non-zero final part (e.g. 1.118.2).
func IsHotfixVersion(name string) bool {
	var b strings.Builder
	for _, r := range name {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		}
	}
	parts := strings.Split(b.String(), ".")
	if len(parts) < 3 {
		return false
	}
	n, err := atoi(parts[len(parts)-1])
	if err != nil {
		return false
	}
	return n > 0
}

// GetAssignee returns the assignee display name (dots replaced by spaces),
// or "Unassigned".
func GetAssignee(issue map[string]any) string {
	a := jx.Obj(jx.Obj(issue["fields"])["assignee"])
	if a == nil {
		return "Unassigned"
	}
	return strings.ReplaceAll(jx.Str(a["displayName"]), ".", " ")
}

// GetExecutor reads the executor custom field, which may hold a single user
// or a list; it returns "" when unset.
func GetExecutor(fields map[string]any, fieldID string) string {
	if fieldID == "" {
		return ""
	}
	v := fields[fieldID]
	if v == nil {
		return ""
	}
	if list, ok := v.([]any); ok {
		if len(list) == 0 {
			return ""
		}
		v = list[0]
	}
	return strings.ReplaceAll(jx.Str(jx.Obj(v)["displayName"]), ".", " ")
}

// ParseChangelog extracts status/assignee transitions from the issue's
// embedded changelog, dropping entries before startDate when it is set.
func ParseChangelog(issue map[string]any, startDate string) []Change {
	changes := []Change{}
	for _, h := range jx.Arr(jx.Obj(issue["changelog"])["histories"]) {
		hm := jx.Obj(h)
		created := cut(jx.Str(hm["created"]), 19)
		if startDate != "" && created < startDate {
			continue
		}
		author := strings.ReplaceAll(jx.Str(jx.Obj(hm["author"])["displayName"]), ".", " ")
		for _, item := range jx.Arr(hm["items"]) {
			im := jx.Obj(item)
			field := jx.Str(im["field"])
			if field != "status" && field != "assignee" {
				continue
			}
			changes = append(changes, Change{
				Date:   created,
				Author: author,
				Field:  field,
				From:   jx.Str(im["fromString"]),
				To:     jx.Str(im["toString"]),
			})
		}
	}
	return changes
}

// FetchFullChangelog pages through the dedicated changelog endpoint when the
// embedded changelog was truncated.
func FetchFullChangelog(c *client.Client, key string) ([]Change, error) {
	all := []any{}
	startAt := 0
	for {
		res, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/changelog?startAt=%d", key, startAt))
		if err != nil {
			return nil, err
		}
		m := jx.Obj(res)
		values := jx.Arr(m["values"])
		all = append(all, values...)
		if jx.Bool(m["isLast"], true) || len(values) == 0 {
			break
		}
		startAt += len(values)
	}
	changes := []Change{}
	for _, h := range all {
		hm := jx.Obj(h)
		created := cut(jx.Str(hm["created"]), 19)
		author := strings.ReplaceAll(jx.Str(jx.Obj(hm["author"])["displayName"]), ".", " ")
		for _, item := range jx.Arr(hm["items"]) {
			im := jx.Obj(item)
			field := jx.Str(im["field"])
			if field != "status" && field != "assignee" {
				continue
			}
			changes = append(changes, Change{
				Date:   created,
				Author: author,
				Field:  field,
				From:   jx.Str(im["fromString"]),
				To:     jx.Str(im["toString"]),
			})
		}
	}
	return changes, nil
}

// DetermineDeveloper returns who first moved the ticket into a dev status
// (chronologically), falling back to the first move into a review status.
func DetermineDeveloper(changes []Change, devStatuses, reviewStatuses map[string]bool) (string, string, bool) {
	chronological := slices.Clone(changes)
	slices.SortStableFunc(chronological, func(a, b Change) int {
		return strings.Compare(a.Date, b.Date)
	})
	for _, ch := range chronological {
		if ch.Field == "status" && devStatuses[ch.To] {
			return ch.Author, ch.Date, true
		}
	}
	for _, ch := range chronological {
		if ch.Field == "status" && reviewStatuses[ch.To] {
			return ch.Author, ch.Date, true
		}
	}
	return "", "", false
}

// DetermineTester returns who moved the ticket from a test status to a
// tested status.
func DetermineTester(changes []Change, testStatuses, testedStatuses map[string]bool) (string, string, bool) {
	chronological := slices.Clone(changes)
	slices.SortStableFunc(chronological, func(a, b Change) int {
		return strings.Compare(a.Date, b.Date)
	})
	for _, ch := range chronological {
		if ch.Field != "status" {
			continue
		}
		if testStatuses[ch.From] && testedStatuses[ch.To] {
			return ch.Author, ch.Date, true
		}
	}
	return "", "", false
}

// Ticket is one row of the digest JSON output.
type Ticket struct {
	Key         string   `json:"key"`
	Project     string   `json:"project"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Assignee    string   `json:"assignee"`
	Developer   *string  `json:"developer"`
	Tester      *string  `json:"tester"`
	Versions    []string `json:"versions"`
	Hotfix      bool     `json:"hotfix"`
	Changelog   []Change `json:"changelog"`
}

// BuildJSON classifies every issue: developer, tester, changelog, hotfix.
// developer/tester are only attributed when the deciding transition happened
// on or after startDate. A failed changelog fetch is returned as an error,
// never silently treated as an empty changelog.
func BuildJSON(c *client.Client, issues []any, startDate, executorFieldID string,
	devS, reviewS, testS, testedS map[string]bool) ([]Ticket, error) {
	tickets := []Ticket{}
	for _, issueAny := range issues {
		issue := jx.Obj(issueAny)
		f := jx.Obj(issue["fields"])
		key := jx.Str(issue["key"])
		project := key
		if i := strings.Index(key, "-"); i > 0 {
			project = key[:i]
		}
		status := jx.Str(jx.Obj(f["status"])["name"])
		versions := []string{}
		for _, v := range jx.Arr(f["fixVersions"]) {
			versions = append(versions, jx.Str(jx.Obj(v)["name"]))
		}
		hotfix := false
		for _, v := range versions {
			if IsHotfixVersion(v) {
				hotfix = true
				break
			}
		}
		desc := ""
		if f["description"] != nil {
			desc = adf.ExtractText(f["description"])
		}
		if len([]rune(desc)) > 300 {
			desc = string([]rune(desc)[:300]) + "..."
		}

		t := Ticket{
			Key:         key,
			Project:     project,
			Summary:     jx.Str(f["summary"]),
			Description: desc,
			Status:      status,
			Assignee:    GetAssignee(issue),
			Versions:    versions,
			Hotfix:      hotfix,
			Changelog:   []Change{},
		}

		if startDate != "" {
			executor := GetExecutor(f, executorFieldID)
			clMeta := jx.Obj(issue["changelog"])
			truncated := jx.Int(clMeta["total"]) > len(jx.Arr(clMeta["histories"]))
			fullChanges := []Change{}
			fetched := false
			ensureFull := func() error {
				if fetched {
					return nil
				}
				changes, err := FetchFullChangelog(c, key)
				if err != nil {
					return fmt.Errorf("changelog of %s: %w", key, err)
				}
				fullChanges, fetched = changes, true
				return nil
			}
			if truncated {
				if err := ensureFull(); err != nil {
					return nil, err
				}
			}
			embedded := ParseChangelog(issue, "")
			// Like the former script: an empty full changelog falls back to
			// the embedded one.
			lookup := func() []Change {
				if len(fullChanges) > 0 {
					return fullChanges
				}
				return embedded
			}

			var developer, devDate string
			haveDev := false
			a, d, ok := DetermineDeveloper(lookup(), devS, reviewS)
			if !ok {
				if err := ensureFull(); err != nil {
					return nil, err
				}
				a, d, ok = DetermineDeveloper(fullChanges, devS, reviewS)
			}
			if ok && executor != "" {
				a = executor
			}
			if ok {
				developer, devDate, haveDev = a, d, true
			}

			tester, testDate := "", ""
			haveTest := false
			a, d, ok = DetermineTester(lookup(), testS, testedS)
			if !ok {
				if err := ensureFull(); err != nil {
					return nil, err
				}
				a, d, ok = DetermineTester(fullChanges, testS, testedS)
			}
			if ok {
				tester, testDate, haveTest = a, d, true
			}

			if haveDev && devDate < startDate {
				developer, haveDev = "", false
			}
			if haveTest && testDate < startDate {
				tester, haveTest = "", false
			}
			if haveDev {
				t.Developer = &developer
			}
			if haveTest {
				t.Tester = &tester
			}

			t.Changelog = ParseChangelog(issue, startDate)
			if len(fullChanges) > 0 {
				t.Changelog = []Change{}
				for _, ch := range fullChanges {
					if ch.Date >= startDate {
						t.Changelog = append(t.Changelog, ch)
					}
				}
			}
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// PersonStats is the per-person rollup in team_stats.
type PersonStats struct {
	Developed int      `json:"developed"`
	Hotfixes  int      `json:"hotfixes"`
	Tested    int      `json:"tested"`
	Reviews   int      `json:"reviews"`
	DevKeys   []string `json:"dev_keys"`
	TestKeys  []string `json:"test_keys"`
}

// TeamStats is the team_stats object of the digest JSON. PerPerson maps a
// display name to PersonStats, ordered by developed count descending.
type TeamStats struct {
	TotalTickets  int         `json:"total_tickets"`
	TotalHotfixes int         `json:"total_hotfixes"`
	TotalReviews  int         `json:"total_reviews"`
	PerPerson     *jx.Ordered `json:"per_person"`
}

// Person returns the stats of name, or nil.
func (s *TeamStats) Person(name string) *PersonStats {
	v, ok := s.PerPerson.Get(name)
	if !ok {
		return nil
	}
	return v.(*PersonStats)
}

// BuildTeamStats rolls tickets up into per-person developed/tested/hotfix/
// review counts. A review is a transition out of a review status.
// total_hotfixes counts every hotfix ticket, attributed or not.
func BuildTeamStats(tickets []Ticket, reviewStatuses map[string]bool) *TeamStats {
	per := jx.NewOrdered()
	get := func(name string) *PersonStats {
		if v, ok := per.Get(name); ok {
			return v.(*PersonStats)
		}
		p := &PersonStats{DevKeys: []string{}, TestKeys: []string{}}
		per.Set(name, p)
		return p
	}
	stats := &TeamStats{TotalTickets: len(tickets), PerPerson: per}
	for _, t := range tickets {
		if t.Hotfix {
			stats.TotalHotfixes++
		}
		if t.Developer != nil && *t.Developer != "" {
			p := get(*t.Developer)
			p.Developed++
			p.DevKeys = append(p.DevKeys, t.Key)
			if t.Hotfix {
				p.Hotfixes++
			}
		}
		if t.Tester != nil && *t.Tester != "" {
			p := get(*t.Tester)
			p.Tested++
			p.TestKeys = append(p.TestKeys, t.Key)
		}
		for _, ch := range t.Changelog {
			if ch.Field != "status" {
				continue
			}
			if reviewStatuses[ch.From] && !reviewStatuses[ch.To] {
				get(ch.Author).Reviews++
			}
		}
	}
	for _, name := range per.Keys() {
		stats.TotalReviews += get(name).Reviews
	}
	per.Sort(func(a, b string) bool {
		return get(a).Developed > get(b).Developed
	})
	return stats
}

// BuildReport renders the plain (non-JSON) markdown digest.
func BuildReport(issues []any) string {
	type entry struct{ status, summary, key, assignee string }
	hotfixes := map[string][]entry{}
	byGroup := map[string][]entry{}

	for _, iss := range issues {
		issue := jx.Obj(iss)
		f := jx.Obj(issue["fields"])
		key := jx.Str(issue["key"])
		project := key
		if i := strings.Index(key, "-"); i > 0 {
			project = key[:i]
		}
		status := jx.Str(jx.Obj(f["status"])["name"])
		summary := jx.Str(f["summary"])
		assignee := GetAssignee(issue)
		hotfixV, releaseV := "", ""
		for _, v := range jx.Arr(f["fixVersions"]) {
			name := jx.Str(jx.Obj(v)["name"])
			if IsHotfixVersion(name) {
				hotfixV = name
			} else {
				releaseV = name
			}
		}
		e := entry{status, summary, key, assignee}
		if hotfixV != "" {
			hotfixes[hotfixV] = append(hotfixes[hotfixV], e)
			continue
		}
		group := project + " — Other"
		if releaseV != "" {
			group = project + " — " + releaseV
		}
		byGroup[group] = append(byGroup[group], e)
	}

	lines := []string{}
	for _, version := range slices.Sorted(maps.Keys(hotfixes)) {
		cleaned := cleanVersion(version)
		lines = append(lines, "### Hotfix Release "+cleaned)
		for _, e := range hotfixes[version] {
			lines = append(lines, fmt.Sprintf("- **%s** — %s (%s, %s)", e.status, e.summary, e.key, e.assignee))
		}
		lines = append(lines, "")
	}
	for _, group := range slices.Sorted(maps.Keys(byGroup)) {
		lines = append(lines, "### "+group)
		for _, e := range byGroup[group] {
			lines = append(lines, fmt.Sprintf("- **%s** — %s (%s, %s)", e.status, e.summary, e.key, e.assignee))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// FetchProjectStatuses returns the sorted workflow status names of projects.
func FetchProjectStatuses(c *client.Client, projects []string) ([]string, error) {
	set := map[string]bool{}
	for _, p := range projects {
		res, err := c.Get("/rest/api/3/project/" + p + "/statuses")
		if err != nil {
			return nil, err
		}
		for _, it := range jx.Arr(res) {
			for _, s := range jx.Arr(jx.Obj(it)["statuses"]) {
				set[jx.Str(jx.Obj(s)["name"])] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(set)), nil
}

// Field is one custom field of `digest fields`.
type Field struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FetchCustomFields returns the custom fields sorted by name.
func FetchCustomFields(c *client.Client) ([]Field, error) {
	res, err := c.Get("/rest/api/3/field")
	if err != nil {
		return nil, err
	}
	result := []Field{}
	for _, f := range jx.Arr(res) {
		fm := jx.Obj(f)
		if !jx.Bool(fm["custom"], false) {
			continue
		}
		result = append(result, Field{ID: jx.Str(fm["id"]), Name: jx.Str(fm["name"])})
	}
	slices.SortStableFunc(result, func(a, b Field) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

func cleanVersion(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cut(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func atoi(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a digit")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
