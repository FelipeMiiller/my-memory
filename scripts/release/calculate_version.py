#!/usr/bin/env python3
"""
calculate_version.py

Calcula o próximo versionamento SemVer (Semantic Versioning 2.0.0) a partir
do histórico de Conventional Commits entre a última Git Tag e o HEAD.
Gera release notes categorizadas e suporta gravação no $GITHUB_OUTPUT.
"""

import argparse
import os
import re
import subprocess
import sys
from typing import Dict, List, Optional, Tuple

RE_CONVENTIONAL = re.compile(
    r"^(?P<type>feat|fix|perf|refactor|docs|test|chore|style|build|ci)(?:\((?P<scope>[a-zA-Z0-9_.-]+)\))?(?P<breaking>!)?:\s*(?P<subject>.+)$",
    re.IGNORECASE,
)

def run_git(args: List[str]) -> Tuple[int, str]:
    """Executa um comando Git e retorna o código de saída e a saída em texto."""
    try:
        proc = subprocess.run(
            ["git"] + args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding="utf-8",
            errors="replace",
        )
        return proc.returncode, proc.stdout.strip()
    except Exception as e:
        return 1, str(e)


def get_latest_tag() -> Optional[str]:
    """Obtém a tag Git anotada mais recente acessível pelo HEAD."""
    code, out = run_git(["describe", "--tags", "--abbrev=0"])
    if code == 0 and out:
        return out.strip()
    return None


def parse_semver(tag: str) -> Tuple[int, int, int]:
    """Faz o parse de uma tag como 'v1.2.3' ou '1.2.3' para tupla (major, minor, patch)."""
    clean = tag.lstrip("v").strip()
    parts = clean.split(".")
    if len(parts) != 3:
        raise ValueError(f"Tag inválida para SemVer: {tag}")
    return int(parts[0]), int(parts[1]), int(parts[2])


def format_semver(major: int, minor: int, patch: int) -> str:
    """Formata a versão com prefixo canônico 'v'."""
    return f"v{major}.{minor}.{patch}"


def parse_commits(commit_range: Optional[str]) -> List[Dict[str, str]]:
    """Obtém e faz o parse da lista de commits no range especificado."""
    args = ["log", "--no-merges", "--format=%H|%s|%b<END_COMMIT>"]
    if commit_range:
        args.append(commit_range)

    code, out = run_git(args)
    if code != 0 or not out:
        return []

    raw_commits = out.split("<END_COMMIT>\n")
    parsed = []

    for raw in raw_commits:
        raw = raw.strip()
        if not raw:
            continue
        parts = raw.split("|", 2)
        sha = parts[0].strip()
        subject = parts[1].strip() if len(parts) > 1 else ""
        body = parts[2].strip() if len(parts) > 2 else ""

        match = RE_CONVENTIONAL.match(subject)
        is_breaking = False
        commit_type = "other"
        scope = ""

        if match:
            commit_type = match.group("type").lower()
            scope = match.group("scope") or ""
            if match.group("breaking") or "BREAKING CHANGE:" in body or "BREAKING-CHANGE:" in body:
                is_breaking = True
        else:
            if "BREAKING CHANGE:" in body or "BREAKING-CHANGE:" in body:
                is_breaking = True

        parsed.append({
            "sha": sha[:7],
            "full_sha": sha,
            "subject": subject,
            "body": body,
            "type": commit_type,
            "scope": scope,
            "is_breaking": is_breaking,
        })

    return parsed


def determine_bump(commits: List[Dict[str, str]]) -> str:
    """Determina o nível de bump SemVer necessário: 'major', 'minor', 'patch' ou 'none'."""
    has_breaking = False
    has_minor = False
    has_patch = False

    for c in commits:
        if c["is_breaking"]:
            has_breaking = True
            break
        ctype = c["type"]
        if ctype == "feat":
            has_minor = True
        elif ctype in ("fix", "perf", "refactor"):
            has_patch = True

    if has_breaking:
        return "major"
    if has_minor:
        return "minor"
    if has_patch:
        return "patch"
    return "none"


def generate_changelog(commits: List[Dict[str, str]], next_tag: str) -> str:
    """Gera notas de release categorizadas no formato Markdown."""
    breaking = []
    features = []
    fixes = []
    other = []

    for c in commits:
        desc = c["subject"]
        if c["scope"]:
            # Destacar escopo
            desc = f"**{c['scope']}**: {desc}"
        line = f"- {desc} ([`{c['sha']}`](https://github.com/FelipeMiiller/my-memory/commit/{c['full_sha']}))"

        if c["is_breaking"]:
            breaking.append(line)
        elif c["type"] == "feat":
            features.append(line)
        elif c["type"] in ("fix", "perf"):
            fixes.append(line)
        else:
            other.append(line)

    sections = [f"## {next_tag}\n"]

    if breaking:
        sections.append("### ⚠️ Breaking Changes")
        sections.extend(breaking)
        sections.append("")

    if features:
        sections.append("### ✨ Features")
        sections.extend(features)
        sections.append("")

    if fixes:
        sections.append("### 🐛 Bug Fixes & Improvements")
        sections.extend(fixes)
        sections.append("")

    if other:
        sections.append("### 📚 Documentation & Maintenance")
        sections.extend(other)
        sections.append("")

    return "\n".join(sections).strip() + "\n"


def main():
    parser = argparse.ArgumentParser(description="Calculador determinístico de versão SemVer via Conventional Commits")
    parser.add_argument("--initial-version", default="v1.0.0", help="Versão inicial se nenhuma tag existir (padrão: v1.0.0)")
    parser.add_argument("--force-bump", choices=["major", "minor", "patch"], help="Força um bump específico ignorando os commits")
    parser.add_argument("--github-output", help="Caminho do arquivo GITHUB_OUTPUT para gravação de variáveis de saída")
    parser.add_argument("--changelog-file", help="Caminho para salvar o Markdown do changelog")
    args = parser.parse_args()

    latest_tag = get_latest_tag()

    if not latest_tag:
        # Repositório virgem de tags: primeira release
        commits = parse_commits(None)
        next_tag = args.initial_version
        bump = "major" if next_tag.startswith("v1.0.0") else "minor"
        should_release = True
        changelog = generate_changelog(commits, next_tag)
    else:
        commits = parse_commits(f"{latest_tag}..HEAD")
        if not commits and not args.force_bump:
            should_release = False
            bump = "none"
            next_tag = latest_tag
            changelog = ""
        else:
            bump = args.force_bump or determine_bump(commits)
            if bump == "none":
                should_release = False
                next_tag = latest_tag
                changelog = ""
            else:
                should_release = True
                major, minor, patch = parse_semver(latest_tag)
                if bump == "major":
                    major += 1
                    minor = 0
                    patch = 0
                elif bump == "minor":
                    minor += 1
                    patch = 0
                elif bump == "patch":
                    patch += 1
                next_tag = format_semver(major, minor, patch)
                changelog = generate_changelog(commits, next_tag)

    print(f"Latest tag:     {latest_tag or 'none'}")
    print(f"Should release: {should_release}")
    print(f"Bump type:      {bump}")
    print(f"Next tag:       {next_tag}")
    print(f"Commits found:  {len(commits)}")

    if args.changelog_file and changelog:
        os.makedirs(os.path.dirname(os.path.abspath(args.changelog_file)), exist_ok=True)
        with open(args.changelog_file, "w", encoding="utf-8") as f:
            f.write(changelog)
        print(f"Changelog salvo em: {args.changelog_file}")

    if args.github_output:
        with open(args.github_output, "a", encoding="utf-8") as f:
            f.write(f"should_release={'true' if should_release else 'false'}\n")
            f.write(f"latest_tag={latest_tag or ''}\n")
            f.write(f"next_tag={next_tag}\n")
            f.write(f"bump_type={bump}\n")
            f.write(f"commit_count={len(commits)}\n")

if __name__ == "__main__":
    main()
