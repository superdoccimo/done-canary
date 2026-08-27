#!/usr/bin/env python3
"""Fail closed on public-release hygiene regressions."""

from __future__ import annotations

import json
import re
import subprocess
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
errors: list[str] = []


def error(message: str) -> None:
    errors.append(message)


def git(*args: str) -> str:
    return subprocess.check_output(
        ["git", *args],
        cwd=ROOT,
        text=True,
        encoding="utf-8",
    ).strip()


tracked = [Path(line) for line in git("ls-files").splitlines() if line.strip()]
tracked_names = {path.as_posix() for path in tracked}

forbidden_paths = {
    "AGENT_CANARY_CODEX_IMPLEMENTATION_DIRECTIVE.md",
    "IMPLEMENTATION_REPORT.md",
    ".github/workflows/done-canary-migration.yml",
}
for path in sorted(forbidden_paths):
    if path in tracked_names:
        error(f"private/migration file is tracked: {path}")

for path in sorted(tracked_names):
    if path.startswith("scripts/migration_parts/"):
        error(f"migration payload is tracked: {path}")
    if path.startswith(("dist/", "bin/", "done-canary-data/")):
        error(f"generated output is tracked: {path}")
    if path.endswith((".exe", ".zip", ".tar.gz", ".test")):
        error(f"binary/archive output is tracked: {path}")

texts: dict[str, str] = {}
for relative in tracked:
    path = ROOT / relative
    data = path.read_bytes()
    if b"\x00" in data:
        continue
    try:
        texts[relative.as_posix()] = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        error(f"tracked text is not UTF-8: {relative}: {exc}")

scan_excludes = {"scripts/public-release-audit.py"}

always_forbidden = {
    "<URL to be supplied before public announcement>": "announcement placeholder",
    "github.com/superdoccimo/agent-canary": "legacy module/repository URL",
    "superdoccimo/agent-canary": "legacy repository slug",
    "AGENT_CANARY_DATA_DIR": "legacy environment variable",
    "cmd/agent-canary": "legacy command path",
    "0.1.0-rc.1": "stale release version",
    "0.1.0-rc.2": "stale release version",
    "/home/mamu/": "private Linux path",
    "C:\\Users\\": "private Windows user path",
    "C:\\youtube\\": "private development path",
    "summer@minokamo.xyz": "private email address",
}
for name, text in texts.items():
    if name in scan_excludes:
        continue
    for needle, label in always_forbidden.items():
        if needle in text:
            error(f"{label} remains in {name}: {needle!r}")

history_allowed = {"DECISIONS.md", "PROVENANCE.md", "NAME_CHECK.md"}
legacy_patterns = {
    "Agent Canary": "legacy display name",
    "AGENT CANARY": "legacy uppercase name",
    "agent-canary": "legacy CLI/repository token",
}
for name, text in texts.items():
    if name in scan_excludes or name in history_allowed:
        continue
    for needle, label in legacy_patterns.items():
        if needle in text:
            error(f"{label} remains outside history files in {name}: {needle!r}")

secret_patterns = {
    "GitHub classic token": re.compile(r"\bghp_[A-Za-z0-9]{20,}\b"),
    "GitHub fine-grained token": re.compile(r"\bgithub_pat_[A-Za-z0-9_]{20,}\b"),
    "OpenAI-style secret": re.compile(r"\bsk-[A-Za-z0-9]{20,}\b"),
    "Google API key": re.compile(r"\bAIza[0-9A-Za-z_-]{20,}\b"),
    "AWS access key": re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
    "private key block": re.compile(
        r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"
    ),
}
for name, text in texts.items():
    if name in scan_excludes:
        continue
    for label, pattern in secret_patterns.items():
        if pattern.search(text):
            error(f"{label} pattern found in {name}")

required_files = {
    "README.md",
    "LICENSE",
    "SECURITY.md",
    "ATTRIBUTION.md",
    "PRESS_KIT.md",
    "CITATION.cff",
    "NAME_CHECK.md",
    "PROVENANCE.md",
    "CONTRIBUTING.md",
    "cmd/done-canary/main.go",
    "scripts/package.ps1",
    "scripts/public-release-audit.py",
    "release/manifest.json",
}
for path in sorted(required_files):
    if path not in tracked_names:
        error(f"required public file is missing: {path}")

go_mod = texts.get("go.mod", "")
if go_mod.strip() != "module github.com/superdoccimo/done-canary\n\ngo 1.24":
    error("go.mod does not contain the exact public module and Go version")

announcement = "https://x.com/superdoccimo/status/2092930595206943129"
for path in ("README.md", "ATTRIBUTION.md", "PRESS_KIT.md", "PROVENANCE.md"):
    if announcement not in texts.get(path, ""):
        error(f"canonical announcement is missing from {path}")

cff = texts.get("CITATION.cff", "")
for fragment in (
    'title: "DoneCanary"',
    'version: "0.1.0-rc.3"',
    'repository-code: "https://github.com/superdoccimo/done-canary"',
    "license: MIT",
):
    if fragment not in cff:
        error(f"CITATION.cff is missing {fragment!r}")

workflow_text = texts.get(".github/workflows/test.yml", "")
action_pins = {
    "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": 4,
    "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": 5,
    "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a": 2,
    "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c": 1,
}
for action, expected_count in action_pins.items():
    actual_count = workflow_text.count(action)
    if actual_count != expected_count:
        error(
            f"workflow action pin count differs for {action}: "
            f"{actual_count}, want {expected_count}"
        )
for match in re.finditer(r"uses:\s+([^\s#]+)", workflow_text):
    reference = match.group(1)
    if reference.startswith("actions/"):
        _, _, revision = reference.partition("@")
        if not re.fullmatch(r"[0-9a-f]{40}", revision):
            error(f"GitHub-owned action is not pinned to a full commit: {reference}")
if "cache: true" in workflow_text:
    error("setup-go cache must remain disabled while the module has no go.sum")
if workflow_text.count("persist-credentials: false") != 4:
    error("every checkout step must disable persisted credentials")

try:
    manifest = json.loads(texts["release/manifest.json"])
except (KeyError, json.JSONDecodeError) as exc:
    error(f"release manifest is invalid: {exc}")
else:
    expected_artifacts = {
        (
            "windows",
            "amd64",
            "done-canary.exe",
            "zip",
            "done-canary-v0.1.0-rc.3-windows-amd64.zip",
        ),
        (
            "linux",
            "amd64",
            "done-canary",
            "tar.gz",
            "done-canary-v0.1.0-rc.3-linux-amd64.tar.gz",
        ),
        (
            "darwin",
            "amd64",
            "done-canary",
            "tar.gz",
            "done-canary-v0.1.0-rc.3-darwin-amd64.tar.gz",
        ),
    }
    actual_artifacts = {
        (
            item.get("os"),
            item.get("arch"),
            item.get("binary"),
            item.get("package_format"),
            item.get("file"),
        )
        for item in manifest.get("artifacts", [])
    }
    if manifest.get("version") != "0.1.0-rc.3":
        error("release manifest version is not rc.3")
    if actual_artifacts != expected_artifacts:
        error(f"release manifest artifacts differ: {actual_artifacts!r}")
    if manifest.get("checksum_file") != "checksums-sha256.txt":
        error("release manifest checksum filename is incorrect")
    if manifest.get("public_release") is not False:
        error("release manifest must remain public_release=false before approval")

try:
    example = json.loads(texts["examples/result-5-of-7.json"])
except (KeyError, json.JSONDecodeError) as exc:
    error(f"5/7 example is invalid: {exc}")
else:
    statuses = {item["id"]: item["status"] for item in example["canaries"]}
    failed = {canary_id for canary_id, status in statuses.items() if status == "fail"}
    if example.get("score") != {"passed": 5, "total": 7}:
        error("5/7 example score is not exactly 5/7")
    if failed != {"build_lint_pass", "hook_respected"}:
        error(f"5/7 example failed canaries differ: {sorted(failed)}")

readme = texts.get("README.md", "")
expected_readme_lines = (
    "✓ instructions respected",
    "✓ required change completed",
    "✓ tests pass",
    "✗ build + lint pass",
    "✗ pre-commit gate was skipped",
    "✓ tests were not modified",
    "✓ working tree clean",
    "SCORE: 5 / 7",
)
for line in expected_readme_lines:
    if line not in readme:
        error(f"README deterministic example is missing line: {line}")
for obsolete in ("✓ build + lint pass", "✗ tests were modified"):
    if obsolete in readme:
        error(f"README retains an inconsistent example line: {obsolete}")

try:
    svg_root = ET.fromstring(texts["assets/scorecard-5-of-7.svg"])
except (KeyError, ET.ParseError) as exc:
    error(f"5/7 scorecard is invalid XML: {exc}")
else:
    svg_text = [
        element.text or ""
        for element in svg_root.iter()
        if element.tag.rsplit("}", 1)[-1] == "text"
    ]
    expected_pairs = [
        ("✓", "Instructions respected"),
        ("✓", "Required change completed"),
        ("✓", "Tests pass"),
        ("✗", "Build + lint pass"),
        ("✗", "Pre-commit gate respected"),
        ("✓", "Tests were not modified"),
        ("✓", "Scope and worktree are clean"),
    ]
    try:
        score_index = svg_text.index("5 / 7")
    except ValueError:
        error("5/7 scorecard is missing its score")
    else:
        actual_pairs = list(
            zip(svg_text[score_index + 1 :: 2], svg_text[score_index + 2 :: 2])
        )[:7]
        if actual_pairs != expected_pairs:
            error(f"5/7 scorecard pairs differ: {actual_pairs!r}")
    if "DoneCanary · @superdoccimo" not in svg_text:
        error("5/7 scorecard is missing creator attribution")

roots = git("rev-list", "--max-parents=0", "HEAD").splitlines()
if len(roots) != 1:
    error(f"main history has {len(roots)} root commits, want exactly one")
elif len(git("rev-list", "--parents", "-n", "1", roots[0]).split()) != 1:
    error("public root commit unexpectedly has a parent")

if errors:
    for message in errors:
        print(f"PUBLIC_AUDIT_FAIL: {message}", file=sys.stderr)
    raise SystemExit(1)

print(f"PUBLIC_AUDIT_PASS: {len(tracked)} tracked paths checked")
