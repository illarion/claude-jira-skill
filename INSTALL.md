# Installation

## Option 1: Plugin install (recommended)

In Claude Code, run:

```
/plugin marketplace add illarion/claude-jira-skill
/plugin install claude-jira-skill
```

## Option 2: Manual install

Clone the repo and symlink the skill into your Claude Code skills directory:

```bash
git clone https://github.com/illarion/claude-jira-skill.git
mkdir -p ~/.claude/skills
ln -s "$(pwd)/claude-jira-skill/skills/jira" ~/.claude/skills/jira
```

## Setup

After installing, go to the root of your work folder, start a Claude Code session and do:
```
/jira login
```

The skill will guide you through authentication via interactive prompts. Follow the prompts to enter your Jira URL, email, and API token.

After successful login, there will be `.jiraskillrc` created in this folder

You can repeat this operation in other folder, and associate them with other jira instances

## Verify

Start a new Claude Code session and type `/jira` — the skill should appear.
