#!/usr/bin/env python3
"""Make arbitrary Jira API calls without exposing credentials."""

import sys
import json
import argparse
from jira_common import load_credentials, jira_request


def main():
    parser = argparse.ArgumentParser(description="Make a Jira API call")
    parser.add_argument("method", help="HTTP method (GET, POST, PUT, DELETE)")
    parser.add_argument("path", help="API path (e.g. /rest/api/3/issue/PROJ-123)")
    parser.add_argument("--data", help="JSON request body")
    args = parser.parse_args()

    jira_url, email, token = load_credentials()

    data = None
    if args.data:
        data = json.loads(args.data)

    result = jira_request(jira_url, email, token, args.method, args.path, data)
    if result is not None:
        print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
