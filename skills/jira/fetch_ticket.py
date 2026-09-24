#!/usr/bin/env python3
"""Fetch a Jira ticket with full details and download image attachments."""

import sys
import os
import json
import urllib.request
import base64
import tempfile
import re
from jira_common import load_credentials, jira_get

ATTACHMENTS_DIR = os.path.join(tempfile.gettempdir(), "jira-attachments")


def download_attachment(email, token, url, dest_path):
    auth = base64.b64encode(f"{email}:{token}".encode()).decode()
    req = urllib.request.Request(url, headers={
        "Authorization": f"Basic {auth}",
    })
    with urllib.request.urlopen(req, timeout=60) as resp:
        if resp.status == 303:
            redirect_url = resp.headers.get("Location")
            req = urllib.request.Request(redirect_url)
            with urllib.request.urlopen(req, timeout=60) as resp2:
                with open(dest_path, "wb") as f:
                    f.write(resp2.read())
            os.chmod(dest_path, 0o600)
            return

        with open(dest_path, "wb") as f:
            f.write(resp.read())
    os.chmod(dest_path, 0o600)


def extract_adf(node, attachment_map=None):
    if not node or not isinstance(node, dict):
        return ""

    t = node.get("type", "")

    if t == "text":
        text = node.get("text", "")
        marks = node.get("marks", [])
        for mark in marks:
            if mark.get("type") == "code":
                text = f"`{text}`"
            if mark.get("type") == "strong":
                text = f"**{text}**"
        return text

    if t == "hardBreak":
        return "\n"

    if t == "mediaSingle" or t == "mediaGroup":
        parts = []
        for child in node.get("content", []):
            parts.append(extract_adf(child, attachment_map))
        return "\n".join(parts)

    if t == "media":
        attrs = node.get("attrs", {})
        alt = attrs.get("alt", "")
        media_id = attrs.get("id", "")
        if attachment_map and alt in attachment_map:
            return f"\n[IMAGE: {alt} -> {attachment_map[alt]}]\n"
        if attachment_map and media_id in attachment_map:
            return f"\n[IMAGE: {attachment_map[media_id]}]\n"
        return f"\n[IMAGE: {alt or media_id}]\n"

    if t == "codeBlock":
        lang = node.get("attrs", {}).get("language", "")
        lines = []
        for child in node.get("content", []):
            lines.append(extract_adf(child, attachment_map))
        code = "".join(lines)
        return f"\n```{lang}\n{code}\n```\n"

    if t == "heading":
        level = node.get("attrs", {}).get("level", 1)
        parts = []
        for child in node.get("content", []):
            parts.append(extract_adf(child, attachment_map))
        text = "".join(parts)
        return f"\n{'#' * level} {text}\n"

    if t == "bulletList" or t == "orderedList":
        items = []
        for i, child in enumerate(node.get("content", [])):
            prefix = f"{i + 1}. " if t == "orderedList" else "- "
            item_text = extract_adf(child, attachment_map).strip()
            items.append(f"{prefix}{item_text}")
        return "\n".join(items) + "\n"

    if t == "listItem":
        parts = []
        for child in node.get("content", []):
            parts.append(extract_adf(child, attachment_map))
        return " ".join(p.strip() for p in parts if p.strip())

    if t == "inlineCard":
        url = node.get("attrs", {}).get("url", "")
        return url

    parts = []
    for child in node.get("content", []):
        parts.append(extract_adf(child, attachment_map))
    result = "".join(parts)

    if t == "paragraph":
        result = result + "\n"

    return result


def main():
    if len(sys.argv) < 2:
        print("Usage: fetch_ticket.py ISSUE_KEY")
        sys.exit(1)

    issue_key = sys.argv[1].upper()
    jira_url, email, token = load_credentials()

    issue = jira_get(jira_url, email, token, f"/rest/api/3/issue/{issue_key}?fields=summary,description,status,assignee,priority,attachment,comment,fixVersions,issuetype")
    fields = issue["fields"]

    os.makedirs(ATTACHMENTS_DIR, exist_ok=True)

    attachments = []
    attachment_map = {}
    for att in (fields.get("attachment") or []):
        is_image = att["mimeType"].startswith("image/")
        local_path = None

        if is_image:
            safe_name = re.sub(r'[^A-Za-z0-9._-]', '_', os.path.basename(att['filename']))
            local_path = os.path.join(ATTACHMENTS_DIR, f"{issue_key}_{safe_name}")
            if not os.path.realpath(local_path).startswith(os.path.realpath(ATTACHMENTS_DIR)):
                local_path = f"DOWNLOAD_FAILED: unsafe filename"
            elif not os.path.exists(local_path):
                try:
                    download_attachment(email, token, att["content"], local_path)
                except Exception as e:
                    local_path = f"DOWNLOAD_FAILED: {e}"

            attachment_map[att["filename"]] = local_path
            media_id = att.get("mediaApiFileId")
            if media_id:
                attachment_map[media_id] = local_path

        attachments.append({
            "filename": att["filename"],
            "mimeType": att["mimeType"],
            "size": att["size"],
            "local_path": local_path,
        })

    desc_text = extract_adf(fields.get("description"), attachment_map).strip()

    comment_data = fields.get("comment", {})
    all_comments = list(comment_data.get("comments") or [])
    total_comments = comment_data.get("total", len(all_comments))
    if total_comments > len(all_comments):
        start_at = len(all_comments)
        while start_at < total_comments:
            page = jira_get(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/comment?startAt={start_at}&maxResults=50")
            batch = page.get("comments", [])
            if not batch:
                break
            all_comments.extend(batch)
            start_at += len(batch)

    comments = []
    for comm in all_comments:
        author = comm.get("author", {}).get("displayName", "Unknown")
        created = comm["created"][:16].replace("T", " ")
        body = extract_adf(comm.get("body"), attachment_map).strip()
        comments.append({
            "author": author,
            "created": created,
            "body": body,
        })

    assignee = fields.get("assignee")
    assignee_name = assignee.get("displayName", "Unassigned") if assignee else "Unassigned"

    priority = fields.get("priority")
    priority_name = priority.get("name", "None") if priority else "None"

    versions = [v["name"] for v in (fields.get("fixVersions") or [])]

    result = {
        "key": issue_key,
        "summary": fields["summary"],
        "status": fields["status"]["name"],
        "assignee": assignee_name,
        "priority": priority_name,
        "type": fields.get("issuetype", {}).get("name", ""),
        "versions": versions,
        "description": desc_text,
        "attachments": attachments,
        "comments": comments,
    }

    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
