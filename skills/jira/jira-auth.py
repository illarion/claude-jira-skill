#!/usr/bin/env python3
"""CLI for setting up Jira authentication in the current directory."""

import argparse
import os
import sys
from getpass import getpass

from jira_common import save_config, DOTFILE


def cmd_login(args):
    parser = argparse.ArgumentParser()
    parser.add_argument("--name", default="")
    parser.add_argument("--url", default="")
    parser.add_argument("--email", default="")
    parser.add_argument("--token", default="")
    parser.add_argument("--projects", default="")
    parser.add_argument("--dir", default="")
    parsed = parser.parse_args(args)

    target_dir = parsed.dir or os.getcwd()
    dotfile = os.path.join(target_dir, DOTFILE)

    if os.path.exists(dotfile):
        print(f"{DOTFILE} already exists in {target_dir}. Remove it first to reconfigure.", file=sys.stderr)
        sys.exit(1)

    interactive = not (parsed.url and parsed.email and parsed.token)

    if interactive:
        name = parsed.name or input("Instance name (e.g. company-prod): ").strip()
        url = parsed.url or input("Jira URL (e.g. https://company.atlassian.net): ").strip().rstrip("/")
        email = parsed.email or input("Email: ").strip()
        token = parsed.token or getpass("API token: ").strip()
        projects = parsed.projects or input("Default projects for digest (comma-separated, optional): ").strip()
    else:
        name = parsed.name
        url = parsed.url.rstrip("/")
        email = parsed.email
        token = parsed.token
        projects = parsed.projects

    if not url or not email or not token:
        print("URL, email, and token are required.", file=sys.stderr)
        sys.exit(1)

    config = {
        "name": name or os.path.basename(target_dir),
        "url": url,
        "email": email,
        "token": token,
        "projects": projects,
    }

    save_config(dotfile, config)

    print(f"Created {DOTFILE} in {target_dir}")
    print(f"Add {DOTFILE} to your .gitignore.")


def cmd_logout(args):
    dotfile = os.path.join(os.getcwd(), DOTFILE)

    if not os.path.exists(dotfile):
        print(f"No {DOTFILE} in this directory.", file=sys.stderr)
        sys.exit(1)

    os.remove(dotfile)
    print(f"Removed {DOTFILE} from {os.getcwd()}")


COMMANDS = {
    "login": cmd_login,
    "logout": cmd_logout,
}


def main():
    if len(sys.argv) < 2 or sys.argv[1] not in COMMANDS:
        print(f"Usage: jira-auth.py {{login|logout}}")
        print("Command line arguments can also be provided for login:")
        print("  --name INSTANCE_NAME")
        print("  --url JIRA_URL")
        print("  --email EMAIL")
        print("  --token API_TOKEN")
        print("  --projects PROJECTS (comma-separated, optional)")
        print("  --dir TARGET_DIR")

        sys.exit(1)

    COMMANDS[sys.argv[1]](sys.argv[2:])


if __name__ == "__main__":
    main()
