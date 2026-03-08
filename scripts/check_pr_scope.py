#!/usr/bin/env python3

import argparse
import fnmatch
import json
import subprocess
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Check PR changed files against label path rules"
    )
    parser.add_argument(
        "--rules", default=".github/pr-scope-rules.json", help="rules JSON path"
    )
    parser.add_argument("--labels", default="", help="comma-separated PR labels")
    parser.add_argument("--base-ref", default="main", help="base branch name")
    parser.add_argument(
        "--changed-files-file",
        default="",
        help="optional file containing changed paths (one per line)",
    )
    return parser.parse_args()


def read_changed_files(base_ref: str, changed_files_file: str) -> list[str]:
    if changed_files_file:
        content = Path(changed_files_file).read_text(encoding="utf-8")
        return [line.strip() for line in content.splitlines() if line.strip()]

    cmd = ["git", "diff", "--name-only", f"origin/{base_ref}...HEAD"]
    result = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if result.returncode != 0:
        print("[pr-scope] failed to compute changed files", file=sys.stderr)
        print(result.stderr.strip(), file=sys.stderr)
        sys.exit(2)

    return [line.strip() for line in result.stdout.splitlines() if line.strip()]


def matches_any(path: str, patterns: list[str]) -> bool:
    return any(fnmatch.fnmatch(path, pattern) for pattern in patterns)


def main() -> int:
    args = parse_args()
    rules_data = json.loads(Path(args.rules).read_text(encoding="utf-8"))

    labels = {
        label.strip().lower() for label in args.labels.split(",") if label.strip()
    }
    changed_files = read_changed_files(args.base_ref, args.changed_files_file)

    if not changed_files:
        print("[pr-scope] no changed files, skip")
        return 0

    always_allowed: list[str] = rules_data.get("always_allowed", [])
    matched_rules = []
    for rule in rules_data.get("rules", []):
        rule_labels = {str(name).strip().lower() for name in rule.get("labels", [])}
        if rule_labels & labels:
            matched_rules.append(rule)

    if not matched_rules:
        print("[pr-scope] no matching rule for current PR labels")
        print(f"labels={sorted(labels)}")
        print("please add at least one label covered by .github/pr-scope-rules.json")
        return 1

    allowed_patterns = list(always_allowed)
    for rule in matched_rules:
        allowed_patterns.extend(rule.get("allowed_paths", []))

    violations = [
        path for path in changed_files if not matches_any(path, allowed_patterns)
    ]
    if violations:
        print("[pr-scope] blocked: changed files exceed allowed scope")
        print(f"labels={sorted(labels)}")
        print("violations:")
        for path in violations:
            print(f"  - {path}")
        return 1

    print("[pr-scope] pass")
    print(f"labels={sorted(labels)}")
    print(f"checked_files={len(changed_files)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
