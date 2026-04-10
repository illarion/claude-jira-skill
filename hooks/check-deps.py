#!/usr/bin/env python3
import sys

if sys.version_info < (3, 8):
    print(f"jira-skill: Python 3.8+ is required, found {sys.version}.", file=sys.stderr)
    sys.exit(1)
