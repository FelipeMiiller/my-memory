#!/usr/bin/env python3
"""
validate_state.py - deterministic completion gate for a feature.

The skill's strongest invariant is "the Verifier is always-on, never prompted;
Execute is not done until validation.md reports PASS." That is prose the model
must remember. This turns it into a checkable pass/fail the closing step runs
automatically, so declaring a feature done without a real Verifier report fails
loudly instead of slipping through.

It does NOT merely check that validation.md exists - a report that exists but is
empty, still holds the template placeholder, or has no evidence would pass a
shallow existence check while proving nothing. This gate requires a real,
filled verdict plus at least one file:line evidence citation.

The verdict is read from the lines that actually carry one: a '## Validation'
heading, a labelled field ('Result:', 'Resultado:', 'Verdict:', 'Veredito:', with
or without bold, any case), or a line that opens with a verdict token
('**PASS** - 25/25 ...'). A report with no such line has no verdict at all - the
document body is never used as a haystack. PASS, FAIL and PARTIAL are three
different verdicts: PARTIAL is never reported as PASS, and a PARTIAL feature
stays red.

Operates only on the .specs/ markdown artifacts (stack- and tool-agnostic). No
dependencies. Run from the project root (the dir that contains .specs), or pass
--root. Meant to be invoked by the skill as the closing gate of Execute, the
same way lessons.py is invoked at distillation - not a manual step.

Usage:
  python3 <skill-dir>/scripts/validate_state.py [feature]
  python3 <skill-dir>/scripts/validate_state.py

  Invoke from the skill directory that ships this script (not the project root).
  Pass --root when cwd is not the project that contains .specs/.

Exit codes: 0 ok, 1 a completed feature is missing a real PASS report,
            2 usage error.
"""

import argparse
import os
import re
import sys

# A file:line citation: a path with an extension, then :<line>. e.g. src/a.ts:42
EVIDENCE_RE = re.compile(r"[\w./-]+\.[A-Za-z0-9]+:\d+")

# A '## Validation' heading.
_HEADING_RE = re.compile(r"^#{1,4}\s*validation\b", re.IGNORECASE)

# A labelled verdict field: 'Result:', 'Resultado:', 'Verdict:', 'Veredito:' -
# with or without bold, any case, optionally with a qualifier word
# ('Verdict final:'). The search runs on the line with its inline code spans
# removed: a label quoted inside backticks is an EXAMPLE of a label, not a
# declaration of a verdict.
_LABEL_RE = re.compile(
    r"\*{0,2}(?:resultado|result|veredito|verdict)(?:\s+\w+)?\*{0,2}\s*:",
    re.IGNORECASE,
)

# An inline code span: `...`.
_CODE_SPAN_RE = re.compile(r"`[^`]*`")

# A line that OPENS with an EMPHASIZED verdict token: '**PASS** - 25/25
# criteria ...'. The token must be emphasised (a declaration, not prose) and
# must be the grammatical subject of the sentence, not the NAME of a field:
# '- **PARTIAL / SPEC-GAP**: 0' is a count in a report's tally, not a verdict,
# so a compound followed by ':' disqualifies the line.
_LEADING_RE = re.compile(
    r"^\s*(?:[*>_#\-]\s*)*\*\*(?P<verdict>PASS|FAIL|PARTIAL)(?![A-Za-z0-9])"
    r"(?P<compound>[A-Za-z0-9 /_-]*)\*\*\s*(?P<after>.*)$"
)

# The unfilled template placeholder: 'PASS | FAIL' (brackets optional).
_PLACEHOLDER_RE = re.compile(r"PASS\s*\|\s*FAIL", re.IGNORECASE)

# The verdict keywords. PARTIAL comes first so that a segment mentioning both
# PARTIAL and PASS is read as PARTIAL, never as PASS.
_KEYWORD_RE = re.compile(r"\b(PARTIAL|PASS|FAIL)\b")


def _feature_dirs(root):
    base = os.path.join(root, ".specs", "features")
    if not os.path.isdir(base):
        return base, []
    dirs = [
        d for d in sorted(os.listdir(base))
        if os.path.isdir(os.path.join(base, d))
    ]
    return base, dirs


def _label_segment(s):
    """Return the text after the first verdict label that is not inside a code span.

    A label quoted inside backticks ('(`**Verdict**: **PARTIAL**`)') is an
    EXAMPLE of a label, not a declaration of a verdict.
    """
    spans = [(m.start(), m.end()) for m in _CODE_SPAN_RE.finditer(s)]
    for m in _LABEL_RE.finditer(s):
        if any(a <= m.start() < b for a, b in spans):
            continue
        return s[m.end():]
    return None


def _segment(line):
    """Return the verdict segment of a candidate line, or None if not a candidate.

    A candidate line is one of:
      (a) a '## Validation' heading;
      (b) a labelled verdict field ('Result:', 'Resultado:', 'Verdict:',
          'Veredito:', with or without bold, any case) that is NOT quoted
          inside a code span - a quoted label is an example, not a declaration;
      (c) a line that OPENS with an EMPHASIZED verdict token ('**PASS** - 25/25
          ...').

    In (c) the token must be the grammatical subject of the sentence, not the
    NAME of a field: '- **PARTIAL / SPEC-GAP**: 0' is a tally in the report's
    counts section, not a verdict declaration.

    The segment is the text the verdict is read from: everything after the
    label's colon, or the leading token itself.
    """
    m = _LEADING_RE.match(line)
    if m:
        # A token that names a field ('- **PARTIAL / SPEC-GAP**: 0') is a tally,
        # not a verdict declaration - only a statement is.
        if not m.group("after").lstrip().startswith(":"):
            return m.group("verdict")
    s = line.strip()
    m = _HEADING_RE.match(s)
    if m:
        rest = s[m.end():]
        return rest.split(":", 1)[1] if ":" in rest else ""
    m = _label_segment(s)
    if m is not None:
        return m
    return None


def _classify(segment):
    """Classify one verdict segment: 'pass', 'fail', 'partial', 'unfilled', or None.

    The FIRST verdict keyword in the segment decides. PARTIAL is a verdict of
    its own: a segment that says 'PARTIAL - PASS for 14 ACs' is PARTIAL, not
    PASS. The literal template placeholder 'PASS | FAIL' is 'unfilled'.
    """
    if _PLACEHOLDER_RE.search(segment):
        return "unfilled"
    m = _KEYWORD_RE.search(segment)
    if not m:
        return None
    return {
        "PARTIAL": "partial",
        "PASS": "pass",
        "FAIL": "fail",
    }[m.group(1).upper()]


def _verdict(text):
    """Return 'pass', 'fail', 'partial', 'unfilled', or None from a validation report.

    Decision rule (deterministic, document order):

      1. Collect the candidate lines - the lines that actually carry a verdict
         (see _segment: '## Validation' heading, labelled 'Result:'/'Resultado:'/
         'Verdict:'/'Veredito:' field, or a line that opens with an EMPHASIZED
         verdict token as the subject of the sentence). A label quoted in a code
         span and a token that names a field are not declarations.
      2. NO candidate line at all -> None ('no verdict'). The whole document is
         never used as the haystack: a prose report that happens to mention
         PASS and FAIL somewhere is not a verdict, and a long report that
         mentions neither is not a template.
      3. Each candidate yields a segment, classified by the first verdict
         keyword it contains (PARTIAL -> partial, PASS -> pass, FAIL -> fail);
         the literal placeholder 'PASS | FAIL' -> unfilled.
      4. File level: if ANY candidate is 'partial' the file is 'partial' -
         PARTIAL has precedence and is never counted as PASS; otherwise the
         FIRST decisive candidate wins.
    """
    decisive = []
    for line in text.splitlines():
        seg = _segment(line)
        if seg is None:
            continue
        cls = _classify(seg)
        if cls:
            decisive.append(cls)
    if not decisive:
        return None
    if "partial" in decisive:
        return "partial"
    return decisive[0]


def _appears_complete(fdir):
    """Conservative completeness heuristic for the cross-check mode.

    A feature 'appears complete' if it already has a validation.md, or if it has
    a tasks.md with at least one task and no unchecked '- [ ]' boxes left. When
    the signal is ambiguous (no tasks.md, Tasks phase skipped), returns False so
    an in-flight feature is never falsely flagged.
    """
    if os.path.exists(os.path.join(fdir, "validation.md")):
        return True
    tasks = os.path.join(fdir, "tasks.md")
    if not os.path.exists(tasks):
        return False
    body = open(tasks, encoding="utf-8", errors="replace").read()
    if not re.search(r"^#{2,4}\s+T\d+\s*:", body, re.MULTILINE):
        return False
    if re.search(r"^\s*-\s*\[\s\]", body, re.MULTILINE):
        return False  # unchecked box remains -> still in progress
    return True


def _check_feature(fdir, name):
    """Return list of error strings for one feature (empty = pass)."""
    errors = []
    vpath = os.path.join(fdir, "validation.md")
    if not os.path.exists(vpath):
        errors.append(
            f"{name}: no validation.md - Execute is not done until the Verifier "
            f"writes it (author != verifier). Dispatch validation before marking done."
        )
        return errors
    text = open(vpath, encoding="utf-8", errors="replace").read()
    verdict = _verdict(text)
    if verdict is None:
        errors.append(f"{name}: validation.md has no PASS/FAIL verdict (a prose-only report does not count)")
    elif verdict == "unfilled":
        errors.append(f"{name}: validation.md verdict is still the template placeholder '[PASS | FAIL]' - not filled")
    elif verdict == "fail":
        errors.append(f"{name}: validation.md verdict is FAIL - route the ranked gaps to fix tasks, then re-verify (feature is not done)")
    elif verdict == "partial":
        errors.append(f"{name}: validation.md verdict is PARTIAL - the feature is not fully verified; close the SPEC-GAP/FAIL acceptance criteria listed in the report before promoting it")
    if verdict == "pass" and not EVIDENCE_RE.search(text):
        errors.append(f"{name}: validation.md is PASS but cites no file:line evidence - evidence-or-zero not satisfied")
    return errors


def _resolve(root, feature):
    base, dirs = _feature_dirs(root)
    if not os.path.isdir(base):
        print(f"validate_state: no {base} directory - nothing to check.")
        return []
    if feature:
        fdir = feature if os.path.isdir(feature) else os.path.join(base, feature)
        if not os.path.isdir(fdir):
            print(f"validate_state: feature not found: {feature}", file=sys.stderr)
            raise SystemExit(2)
        return [(fdir, os.path.basename(fdir.rstrip("/")))]
    if len(dirs) == 1:
        return [(os.path.join(base, dirs[0]), dirs[0])]
    if not dirs:
        print("validate_state: no features under .specs/features/ - nothing to check.")
        return []
    # Cross-check mode: only features that appear complete.
    picked = [(os.path.join(base, d), d) for d in dirs if _appears_complete(os.path.join(base, d))]
    if not picked:
        print("validate_state: no completed feature detected (all in progress) - nothing to gate.")
    return picked


def main(argv=None):
    p = argparse.ArgumentParser(prog="validate_state.py", description="Deterministic completion gate: a done feature must have a real PASS validation report.")
    p.add_argument("feature", nargs="?", default=None, help="Feature dir or name (default: sole feature, else cross-check all completed)")
    p.add_argument("--root", default=".", help="Project root containing .specs/ (default: current dir)")
    args = p.parse_args(argv)
    root = os.path.abspath(args.root)

    targets = _resolve(root, args.feature)
    all_errors = []
    for fdir, name in targets:
        all_errors += _check_feature(fdir, name)

    for e in all_errors:
        print(f"  ERROR {e}")
    n = len(all_errors)
    checked = ", ".join(name for _, name in targets) or "(none)"
    print(f"\nvalidate_state: {n} error(s) across [{checked}]")
    return 1 if n else 0


if __name__ == "__main__":
    raise SystemExit(main())
