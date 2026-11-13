#!/usr/bin/env python3
"""Block local-only research and adaptation-only material from public releases."""

import argparse
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile


# Split literal names so the validator does not match its own source code.
NAME_ALTERNATIVES = (
    "Qu" + "inn",
    "Glo" + "ria",
    "Mead" + "ows",
    "Si" + "ms",
    "Raid" + "ers?",
    "Flame" + "keepers?",
    "Syn" + "drome",
    "Com" + "pound",
)

RULES = (
    (
        "adaptation-only person, organization, or substance",
        re.compile(r"\b(?:" + "|".join(NAME_ALTERNATIVES) + r")\b", re.IGNORECASE),
    ),
    (
        "adaptation-only full name",
        re.compile(
            r"\b(?:Ron\s+Tucker|Russell\s+Conroy|Paul\s+Billings|"
            r"Patrick\s+Kennedy|Danny\s+Bly|Pete\s+Nichols)\b",
            re.IGNORECASE,
        ),
    ),
    (
        "adaptation-only named entity",
        re.compile(
            r"\b(?:Silo\s+Council|Mission\s+Diary|"
            r"Amazing\s+Adventures\s+in\s+Georgia)\b",
            re.IGNORECASE,
        ),
    ),
    (
        "adaptation-only command entity",
        re.compile(r"\b(?:The|the)\s+" + "Leg" + r"acy\b(?!\s+of\s+Frostpunk)"),
    ),
    (
        "adaptation-only command channel",
        re.compile(r"\b" + "Leg" + r"acy\s+(?:voice|terminal|room|vault)\b", re.IGNORECASE),
    ),
    (
        "adaptation-only watcher role",
        re.compile(r"\bWatch" + r"er\b"),
    ),
    (
        "adaptation-only chronology",
        re.compile(r"\b(?:(?:Year|Y)\s*)?(?:109|143)\s*AR\b", re.IGNORECASE),
    ),
    (
        "adaptation-only relic or cipher plot",
        re.compile(
            "密码" + r"信|情书" + r"转密码|第\s*77\s*页|"
            + r"Amazing\s+Adventures\s+in\s+Georgia",
            re.IGNORECASE,
        ),
    ),
    (
        "adaptation-only illness or slogan",
        re.compile("清洁" + "生活是保障|症状" + "海报"),
    ),
    (
        "adaptation-only termination craft",
        re.compile(
            r"Safeguard\s+Protocol|"
            + "实体" + "管道|毒气" + "管道|组" + "中心堡|中心堡" + "拓扑",
            re.IGNORECASE,
        ),
    ),
    (
        "adaptation-only external pursuit",
        re.compile(
            r"drone\s+pursuit|"
            + "无人机" + "追杀|外部" + "追杀|"
            + ("外" + "逃者") + r".{0,20}(?:" + ("追" + "杀") + "|追击)|(?:"
            + ("追" + "杀") + r"|追击).{0,20}" + ("外" + "逃者"),
            re.IGNORECASE,
        ),
    ),
    (
        "removed adaptation-derived event",
        re.compile(r"\b" + "silo1_drone" + r"_scan\b", re.IGNORECASE),
    ),
    (
        "adaptation-only active silo list",
        re.compile(r"1\s*[/、,，]\s*14\s*[/、,，]\s*18\s*[/、,，]\s*19\s*[/、,，]\s*31\s*[/、,，]\s*42\s*[/、,，]\s*44"),
    ),
    (
        "adaptation-specific external reference",
        re.compile(
            r"Apple\s*TV\+?|Wool\s+Books\s+vs\s+Silo\s+TV\s+Show|"
            r"silo\.fandom\.com",
            re.IGNORECASE,
        ),
    ),
)

LOCAL_ONLY_FILES = {
    "docs/research/01-worldbuilding.md",
    "docs/research/02-timeline.md",
    "docs/research/03-characters-and-factions.md",
    "docs/research/04-books-vs-show.md",
    "docs/research/05-game-adaptation-motifs.md",
    "docs/research/06-silo40-event-plan.md",
}

LOCAL_ONLY_PREFIXES = (
    "docs/research/local/",
    "docs/local/",
    "references/local/",
    "frontend/src/references/",
    "research/",
    "notes/",
    ".local-references/",
    "local-references/",
)


def is_local_only(relative_path):
    return relative_path in LOCAL_ONLY_FILES or relative_path.startswith(LOCAL_ONLY_PREFIXES)


def tracked_files(root):
    result = subprocess.run(
        ["git", "ls-files", "-z"],
        cwd=root,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return [root / item.decode("utf-8") for item in result.stdout.split(b"\0") if item]


def all_files(root):
    files = []
    for directory, dirnames, filenames in os.walk(root):
        dirnames[:] = sorted(name for name in dirnames if name != ".git")
        for filename in sorted(filenames):
            files.append(Path(directory) / filename)
    return files


def scan(root, use_all_files=False):
    findings = []
    files = all_files(root) if use_all_files or not (root / ".git").exists() else tracked_files(root)
    for path in files:
        relative_path = path.relative_to(root).as_posix()
        if is_local_only(relative_path):
            findings.append(f"{relative_path}: local-only path must not enter a release")
            continue
        if path.is_symlink():
            findings.append(f"{relative_path}: symbolic links are not allowed in a release scan")
            continue
        if not path.is_file():
            continue
        try:
            text = path.read_bytes().decode("utf-8", errors="ignore")
        except OSError as exc:
            findings.append(f"{relative_path}: cannot read file: {exc}")
            continue
        for line_number, line in enumerate(text.splitlines(), start=1):
            for label, pattern in RULES:
                if pattern.search(line):
                    findings.append(f"{relative_path}:{line_number}: {label}")
        for label, pattern in RULES:
            if pattern.search(relative_path):
                findings.append(f"{relative_path}: path contains {label}")
    return files, findings


def self_test():
    with tempfile.TemporaryDirectory() as temporary_directory:
        root = Path(temporary_directory)
        (root / "README.md").write_text("Original strategy game\n", encoding="utf-8")
        _, clean_findings = scan(root, use_all_files=True)
        if clean_findings:
            raise AssertionError(f"clean fixture failed: {clean_findings}")
        forbidden_name = "Flame" + "keepers"
        (root / "bad.md").write_text(forbidden_name + "\n", encoding="utf-8")
        _, bad_findings = scan(root, use_all_files=True)
        if not any("bad.md" in finding for finding in bad_findings):
            raise AssertionError("forbidden-name fixture was not detected")
        (root / "bad.md").unlink()
        local_file = root / "docs" / "research" / "local" / "note.md"
        local_file.parent.mkdir(parents=True)
        local_file.write_text("original notes\n", encoding="utf-8")
        _, path_findings = scan(root, use_all_files=True)
        if not any("local-only path" in finding for finding in path_findings):
            raise AssertionError("local-only path fixture was not detected")
    print("PASS: scanner self-test")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    parser.add_argument("--all", action="store_true", help="scan the full worktree instead of tracked files")
    parser.add_argument("--self-test", action="store_true", help="verify that clean and blocked fixtures behave correctly")
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return 0
    root = args.root.resolve()
    if not (root / "README.md").is_file():
        print("FAIL: repository root does not contain README.md", file=sys.stderr)
        return 2
    files, findings = scan(root, use_all_files=args.all)
    if findings:
        for finding in findings:
            print(f"FAIL: {finding}", file=sys.stderr)
        print(f"FAIL: {len(findings)} public-content violation(s)", file=sys.stderr)
        return 1
    scope = "full worktree" if args.all or not (root / ".git").exists() else "tracked files"
    print(f"PASS: scanned {len(files)} files ({scope}); no blocked content found")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
