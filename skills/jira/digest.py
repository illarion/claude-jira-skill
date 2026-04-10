#!/usr/bin/env python3
"""Jira digest report generator."""

import sys
import os
import json
import time
import urllib.request
import urllib.parse
import urllib.error
import base64
import re
import argparse
from collections import defaultdict
from jira_common import load_credentials_full as load_credentials


def _urlopen_with_retry(req, max_attempts=5):
    for attempt in range(max_attempts):
        try:
            return urllib.request.urlopen(req, timeout=30)
        except urllib.error.HTTPError as e:
            if e.code != 429 or attempt == max_attempts - 1:
                raise
            retry_after = int(e.headers.get("Retry-After", 0))
            delay = max(retry_after, 2 ** attempt)
            print(f"Rate limited, retrying in {delay}s...", file=sys.stderr)
            time.sleep(delay)


def jira_search(jira_url, email, token, jql, fields, max_results=50, expand=None):
    issues = []
    next_token = None

    while True:
        query = {
            "jql": jql,
            "maxResults": max_results,
            "fields": fields,
        }
        if expand:
            query["expand"] = expand
        if next_token:
            query["nextPageToken"] = next_token
        params = urllib.parse.urlencode(query)
        url = f"{jira_url}/rest/api/3/search/jql?{params}"
        auth = base64.b64encode(f"{email}:{token}".encode()).decode()

        req = urllib.request.Request(url, headers={
            "Accept": "application/json",
            "Authorization": f"Basic {auth}",
        })
        with _urlopen_with_retry(req) as resp:
            data = json.loads(resp.read())

        batch = data.get("issues", [])
        issues.extend(batch)

        if not batch or data.get("isLast", True):
            break
        next_token = data.get("nextPageToken")
        if not next_token:
            break

    return issues


def is_hotfix_version(version_name):
    cleaned = re.sub(r"[^\d.]", "", version_name)
    parts = cleaned.split(".")
    if len(parts) < 3:
        return False
    try:
        return int(parts[-1]) > 0
    except ValueError:
        return False


def get_assignee(issue):
    assignee = issue["fields"].get("assignee")
    if not assignee:
        return "Unassigned"
    return assignee.get("displayName", "").replace(".", " ")


def extract_adf_text(node):
    if not node or not isinstance(node, dict):
        return ""
    parts = []
    if node.get("type") == "text":
        parts.append(node.get("text", ""))
    for child in node.get("content", []):
        parts.append(extract_adf_text(child))
    return " ".join(parts).strip()


def parse_changelog(issue, start_date=None):
    histories = issue.get("changelog", {}).get("histories", [])
    changes = []
    for history in histories:
        created = history["created"][:19]
        if start_date and created < start_date:
            continue
        author = history.get("author", {}).get("displayName", "").replace(".", " ")
        for item in history.get("items", []):
            if item["field"] not in ("status", "assignee"):
                continue
            changes.append({
                "date": created,
                "author": author,
                "field": item["field"],
                "from": item.get("fromString", ""),
                "to": item.get("toString", ""),
            })
    return changes


def get_executor(issue, executor_field_id):
    if not executor_field_id:
        return None
    executor = issue["fields"].get(executor_field_id)
    if not executor:
        return None
    if isinstance(executor, list):
        if not executor:
            return None
        executor = executor[0]
    return executor.get("displayName", "").replace(".", " ")


def fetch_full_changelog(jira_url, email, token, issue_key):
    auth = base64.b64encode(f"{email}:{token}".encode()).decode()
    all_histories = []
    start_at = 0

    while True:
        url = f"{jira_url}/rest/api/3/issue/{issue_key}/changelog?startAt={start_at}"
        req = urllib.request.Request(url, headers={
            "Accept": "application/json",
            "Authorization": f"Basic {auth}",
        })
        with _urlopen_with_retry(req) as resp:
            data = json.loads(resp.read())

        values = data.get("values", [])
        all_histories.extend(values)

        if data.get("isLast", True) or not values:
            break
        start_at += len(values)

    changes = []
    for history in all_histories:
        created = history["created"][:19]
        author = history.get("author", {}).get("displayName", "").replace(".", " ")
        for item in history.get("items", []):
            if item["field"] not in ("status", "assignee"):
                continue
            changes.append({
                "date": created,
                "author": author,
                "field": item["field"],
                "from": item.get("fromString", ""),
                "to": item.get("toString", ""),
            })
    return changes


def fetch_project_statuses(jira_url, email, token, projects):
    all_statuses = set()
    auth = base64.b64encode(f"{email}:{token}".encode()).decode()
    for project in projects:
        url = f"{jira_url}/rest/api/3/project/{project}/statuses"
        req = urllib.request.Request(url, headers={
            "Accept": "application/json",
            "Authorization": f"Basic {auth}",
        })
        with _urlopen_with_retry(req) as resp:
            issue_types = json.loads(resp.read())
        for it in issue_types:
            for s in it.get("statuses", []):
                all_statuses.add(s["name"])
    return sorted(all_statuses)


def determine_developer(changes, dev_statuses, review_statuses):
    chronological = sorted(changes, key=lambda c: c["date"])

    for change in chronological:
        if change["field"] == "status" and change["to"] in dev_statuses:
            return change["author"], change["date"]

    for change in chronological:
        if change["field"] == "status" and change["to"] in review_statuses:
            return change["author"], change["date"]

    return None, None


def determine_tester(changes, test_statuses, tested_statuses):
    chronological = sorted(changes, key=lambda c: c["date"])

    for change in chronological:
        if change["field"] != "status":
            continue
        if change["from"] in test_statuses and change["to"] in tested_statuses:
            return change["author"], change["date"]

    return None, None


def build_team_stats(tickets, review_statuses=None):
    review_statuses = review_statuses or set()
    per_dev = defaultdict(lambda: {"developed": 0, "hotfixes": 0, "tested": 0, "reviews": 0, "dev_keys": [], "test_keys": []})

    for t in tickets:
        developer = t["developer"]
        if developer:
            per_dev[developer]["developed"] += 1
            per_dev[developer]["dev_keys"].append(t["key"])
            if t["hotfix"]:
                per_dev[developer]["hotfixes"] += 1

        tester = t.get("tester")
        if tester:
            per_dev[tester]["tested"] += 1
            per_dev[tester]["test_keys"].append(t["key"])

        for c in t["changelog"]:
            if c["field"] != "status":
                continue
            if c["from"] in review_statuses and c["to"] not in review_statuses:
                per_dev[c["author"]]["reviews"] += 1

    total_tickets = len(tickets)
    total_hotfixes = sum(1 for t in tickets if t["hotfix"])
    total_reviews = sum(w["reviews"] for w in per_dev.values())

    return {
        "total_tickets": total_tickets,
        "total_hotfixes": total_hotfixes,
        "total_reviews": total_reviews,
        "per_person": {k: v for k, v in sorted(per_dev.items(), key=lambda x: -x[1]["developed"])},
    }


def build_json(issues, start_date=None, executor_field_id=None,
               dev_statuses=None, review_statuses=None, test_statuses=None, tested_statuses=None,
               jira_url=None, email=None, token=None):
    dev_statuses = dev_statuses or set()
    review_statuses = review_statuses or set()
    test_statuses = test_statuses or set()
    tested_statuses = tested_statuses or set()
    result = []
    for issue in issues:
        f = issue["fields"]
        status = f["status"]["name"]
        versions = [v["name"] for v in (f.get("fixVersions") or [])]
        key = issue["key"]
        project = key.split("-")[0]
        assignee = get_assignee(issue)

        hotfix = any(is_hotfix_version(v) for v in versions)

        desc = extract_adf_text(f.get("description")) if f.get("description") else ""
        if len(desc) > 300:
            desc = desc[:300] + "..."

        developer = None
        dev_date = None
        tester = None
        test_date = None
        changelog = []
        if start_date:
            executor = get_executor(issue, executor_field_id)
            changelog_meta = issue.get("changelog", {})
            truncated = changelog_meta.get("total", 0) > len(changelog_meta.get("histories", []))
            full_changes = None
            if truncated and jira_url:
                full_changes = fetch_full_changelog(jira_url, email, token, key)
            embedded_changes = parse_changelog(issue)

            if executor:
                changes_for_lookup = full_changes or embedded_changes
                _, dev_date = determine_developer(changes_for_lookup, dev_statuses, review_statuses)
                if not dev_date and not full_changes and jira_url:
                    full_changes = fetch_full_changelog(jira_url, email, token, key)
                    _, dev_date = determine_developer(full_changes, dev_statuses, review_statuses)
                if dev_date:
                    developer = executor
            else:
                changes_for_lookup = full_changes or embedded_changes
                developer, dev_date = determine_developer(changes_for_lookup, dev_statuses, review_statuses)
                if not developer and not full_changes and jira_url:
                    full_changes = fetch_full_changelog(jira_url, email, token, key)
                    developer, dev_date = determine_developer(full_changes, dev_statuses, review_statuses)

            changes_for_lookup = full_changes or embedded_changes
            tester, test_date = determine_tester(changes_for_lookup, test_statuses, tested_statuses)
            if not tester and not full_changes:
                if jira_url:
                    full_changes = fetch_full_changelog(jira_url, email, token, key)
                if full_changes:
                    tester, test_date = determine_tester(full_changes, test_statuses, tested_statuses)

            if developer and dev_date and dev_date < start_date:
                developer = None
            if tester and test_date and test_date < start_date:
                tester = None

            if full_changes:
                changelog = [c for c in full_changes if not start_date or c["date"] >= start_date]
            else:
                changelog = parse_changelog(issue, start_date)

        result.append({
            "key": key,
            "project": project,
            "summary": f["summary"],
            "description": desc,
            "status": status,
            "assignee": assignee,
            "developer": developer,
            "tester": tester,
            "versions": versions,
            "hotfix": hotfix,
            "changelog": changelog,
        })

    return result


def build_report(issues):
    hotfixes = defaultdict(list)
    by_project_version = defaultdict(list)

    for issue in issues:
        f = issue["fields"]
        status = f["status"]["name"]
        versions = [v["name"] for v in (f.get("fixVersions") or [])]
        key = issue["key"]
        project = key.split("-")[0]
        summary = f["summary"]
        assignee = get_assignee(issue)

        hotfix_v = None
        release_v = None
        for v in versions:
            if is_hotfix_version(v):
                hotfix_v = v
            else:
                release_v = v

        if hotfix_v:
            hotfixes[hotfix_v].append((status, summary, key, assignee))
            continue

        group_key = f"{project} — {release_v}" if release_v else f"{project} — Other"
        by_project_version[group_key].append((status, summary, key, assignee))

    lines = []

    for version in sorted(hotfixes.keys()):
        cleaned = re.sub(r"[^\d.]", "", version)
        lines.append(f"### Hotfix Release {cleaned}")
        for label, summary, key, assignee in hotfixes[version]:
            lines.append(f"- **{label}** — {summary} ({key}, {assignee})")
        lines.append("")

    for group in sorted(by_project_version.keys()):
        lines.append(f"### {group}")
        for label, summary, key, assignee in by_project_version[group]:
            lines.append(f"- **{label}** — {summary} ({key}, {assignee})")
        lines.append("")

    return "\n".join(lines)


def fetch_custom_fields(jira_url, email, token):
    url = f"{jira_url}/rest/api/3/field"
    auth = base64.b64encode(f"{email}:{token}".encode()).decode()
    req = urllib.request.Request(url, headers={
        "Accept": "application/json",
        "Authorization": f"Basic {auth}",
    })
    with _urlopen_with_retry(req) as resp:
        fields = json.loads(resp.read())
    result = []
    for f in fields:
        if not f.get("custom"):
            continue
        result.append({"id": f["id"], "name": f["name"]})
    return sorted(result, key=lambda x: x["name"])


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "fields":
        jira_url, email, token, _ = load_credentials()
        fields = fetch_custom_fields(jira_url, email, token)
        print(json.dumps(fields, indent=2))
        return

    if len(sys.argv) > 1 and sys.argv[1] == "statuses":
        jira_url, email, token, default_projects = load_credentials()
        projects = sys.argv[2].split(",") if len(sys.argv) > 2 else default_projects
        if not projects:
            print("No projects specified and none configured in .jiraskillrc", file=sys.stderr)
            sys.exit(1)
        statuses = fetch_project_statuses(jira_url, email, token, projects)
        print(json.dumps(statuses, indent=2))
        return

    parser = argparse.ArgumentParser(description="Jira digest report generator")
    parser.add_argument("start_date", help="Start date (YYYY-MM-DD)")
    parser.add_argument("projects", nargs="?", default=None, help="Comma-separated project keys")
    parser.add_argument("--json", dest="output_json", action="store_true", help="Output JSON format")
    parser.add_argument("--output-file", help="Write full JSON to file, print summary to stdout")
    parser.add_argument("--dev-statuses", default="", help="Comma-separated dev statuses")
    parser.add_argument("--review-statuses", default="", help="Comma-separated review statuses")
    parser.add_argument("--test-statuses", default="", help="Comma-separated test statuses")
    parser.add_argument("--tested-statuses", default="", help="Comma-separated tested statuses")
    parser.add_argument("--executor-field-id", help="Custom field ID for executor")
    args = parser.parse_args()

    jira_url, email, token, default_projects = load_credentials()

    projects = args.projects.split(",") if args.projects else default_projects
    if not projects:
        print("No projects specified and none configured in .jiraskillrc", file=sys.stderr)
        sys.exit(1)

    dev_statuses = set(args.dev_statuses.split(",")) if args.dev_statuses else set()
    review_statuses = set(args.review_statuses.split(",")) if args.review_statuses else set()
    test_statuses = set(args.test_statuses.split(",")) if args.test_statuses else set()
    tested_statuses = set(args.tested_statuses.split(",")) if args.tested_statuses else set()

    project_list = ", ".join(projects)
    jql = f'project in ({project_list}) AND updated >= "{args.start_date}" ORDER BY project, updated DESC'

    fields = "summary,status,assignee,priority,updated,issuetype,resolution,fixVersions"
    if args.output_json:
        fields += ",description"
        if args.executor_field_id:
            fields += "," + args.executor_field_id

    expand = "changelog" if args.output_json else None
    issues = jira_search(jira_url, email, token, jql, fields, expand=expand)

    if args.output_json:
        tickets = build_json(issues, start_date=args.start_date, executor_field_id=args.executor_field_id,
                             dev_statuses=dev_statuses, review_statuses=review_statuses,
                             test_statuses=test_statuses, tested_statuses=tested_statuses,
                             jira_url=jira_url, email=email, token=token)
        team_stats = build_team_stats(tickets, review_statuses=review_statuses)
        output = {
            "tickets": tickets,
            "team_stats": team_stats,
        }
        full_json = json.dumps(output, indent=2)

        if args.output_file:
            with open(args.output_file, "w") as f:
                f.write(full_json)
            os.chmod(args.output_file, 0o600)
            print(json.dumps(team_stats, indent=2))
        else:
            print(full_json)
        return

    report = build_report(issues)
    print(f"## Team Digest — week of {args.start_date}\n")
    print(report)


if __name__ == "__main__":
    main()
