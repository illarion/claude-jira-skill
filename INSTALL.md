# Installation

## Option 1: Plugin install (recommended)

In Claude Code, run:

```
/plugin install illarion/claude-jira-skill
```

## Option 2: Manual install

Clone the repo and symlink the skill into your Claude Code skills directory:

```bash
git clone https://github.com/illarion/claude-jira-skill.git
mkdir -p ~/.claude/skills
ln -s "$(pwd)/claude-jira-skill/skills/jira" ~/.claude/skills/jira
```

## Setup

After installing, start a Claude Code session and mention a Jira ticket (e.g. `PROJ-1234`). The skill will guide you through authentication via interactive prompts.

Alternatively, configure manually from your company/project root directory:

```bash
python3 ~/.claude/skills/jira/jira-auth.py login --name prod --dir /path/to/company
```

Follow the prompts to enter your Jira URL, email, and API token.

## Verify

Start a new Claude Code session and type `/jira` — the skill should appear.
