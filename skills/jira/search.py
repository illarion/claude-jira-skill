#!/usr/bin/env python3
"""Search Jira issues with JQL."""

import sys
import json
import argparse
import urllib.parse
from jira_common import load_credentials, jira_get

DEFAULT_FIELDS = "summary,status,assignee,priority"


def jira_search(jira_url, email, token, jql, fields, max_results):
    issues = []
    next_token = None

    while True:
        query = {
            "jql": jql,
            "maxResults": min(max_results - len(issues), 50),
            "fields": fields,
        }
        if next_token:
            query["nextPageToken"] = next_token
        params = urllib.parse.urlencode(query)
        data = jira_get(jira_url, email, token, f"/rest/api/3/search/jql?{params}")

        batch = data.get("issues", [])
        issues.extend(batch)

        if len(issues) >= max_results:
            break
        if not batch or data.get("isLast", True):
            break
        next_token = data.get("nextPageToken")
        if not next_token:
            break

    return issues[:max_results]


def simplify(issue, field_list):
    fields = issue["fields"]
    result = {"key": issue["key"]}
    for f in field_list:
        if f == "summary":
            result["summary"] = fields.get("summary", "")
        elif f == "status":
            result["status"] = (fields.get("status") or {}).get("name", "")
        elif f == "assignee":
            assignee = fields.get("assignee")
            result["assignee"] = assignee.get("displayName", "") if assignee else "Unassigned"
        elif f == "priority":
            priority = fields.get("priority")
            result["priority"] = priority.get("name", "") if priority else "None"
        else:
            result[f] = fields.get(f, "")
    return result


def main():
    parser = argparse.ArgumentParser(description="Search Jira issues with JQL")
    parser.add_argument("jql", help="JQL query string")
    parser.add_argument("--fields", default=DEFAULT_FIELDS, help="Comma-separated fields")
    parser.add_argument("--max", type=int, default=50, help="Maximum results")
    args = parser.parse_args()

    jira_url, email, token = load_credentials()
    field_list = [f.strip() for f in args.fields.split(",")]
    issues = jira_search(jira_url, email, token, args.jql, args.fields, args.max)
    results = [simplify(issue, field_list) for issue in issues]
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
