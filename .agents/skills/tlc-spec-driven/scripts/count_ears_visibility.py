#!/usr/bin/env python3
"""
count_ears_visibility.py - mede a cegueira do gate de EARS (ISSUE-050).

O gate `validate_spec.py` so considera "visivel" o AC que ele julga. Uma
spec pode ter ACs que o gate nunca ve: o bloco de Acceptance Criteria
acabava numa linha em branco e o resto do bloco sumia da contagem.

O script compara tres numeros:
  vistos     - os ACs que o gate de fato julga (a colecao viva do
               validate_spec.py, importada daqui - se ela regredir, o
               numero cai junto)
  presentes - os ACs que existem nos blocos, pela regra da spec: o bloco
               termina em heading (`#{1,3}`) ou em linha iniciada por `**`;
               linha vazia e espacamento, nunca estrutura
  cegos     - os ACs presentes que o gate nao ve

Sai com codigo != 0 quando ha bloco cego - o script serve de gate.

Uso:
  python3 <skill-dir>/scripts/count_ears_visibility.py [--root DIR] [--legacy]

  --root    raiz do projeto que contem .specs/ (default: cwd)
  --legacy  mede com a regra antiga (linha em branco encerra o bloco) e
            reproduz o baseline da ISSUE-050
"""

import argparse
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from validate_spec import classify_ears, collect_ac_items  # noqa: E402

HEADER_RE = re.compile(r"^(?:#{1,3}\s+|\*{0,2})Acceptance Criteria:?\*{0,2}\s*:?\s*$", re.IGNORECASE)
ITEM_RE = re.compile(r"^\s*\d+\.\s+(.*)$")
PLACEHOLDER_RE = re.compile(r"^\s*\[.+\]\s*$")
# Cabecalho que parece de AC mas que o validador nao reconhece.
AC_LIKE_RE = re.compile(r"^(?:#{1,6}\s+|\*+\s*)?Acceptance Criteria\b", re.IGNORECASE)

PROBE_LINES = [
    "**Acceptance Criteria**:",
    "",
    "1. O sistema cadastra o usuario sem confirmacao.",
]


def _items_of(lines, ends_on_blank_line):
    """Blocos de AC reconhecidos, pela regra dada.

    Devolve [(linha do cabecalho, [(linha, criterio), ...]), ...].

    ends_on_blank_line=True reproduz a regra antiga (a linha em branco
    encerra o bloco) - e o baseline da ISSUE-050.
    """
    blocks = []
    in_ac = False
    cur = None
    for i, ln in enumerate(lines, start=1):
        stripped = ln.strip()
        if HEADER_RE.match(stripped):
            in_ac = True
            cur = (i, [])
            blocks.append(cur)
            continue
        if in_ac:
            m = ITEM_RE.match(ln)
            if m:
                item = m.group(1).strip()
                if not PLACEHOLDER_RE.match(item):
                    cur[1].append((i, item))
            elif (
                (ends_on_blank_line and stripped == "")
                or re.match(r"^#{1,3}\s", ln)
                or stripped.startswith("**")
            ):
                in_ac = False
                cur = None
    return blocks


def _unrecognized_blocks(lines):
    """Blocos de AC sob cabecalho que o validador nao reconhece.

    Nao e cegueira da ISSUE-050 (e cegueira de cabecalho): o validador nunca
    abre o bloco, porque o cabecalho nao casa com o regex dele. Fica fora do
    gate - e reportado so como diagnostico.
    """
    blocks = []
    in_ac = False
    cur = None
    for i, ln in enumerate(lines, start=1):
        stripped = ln.strip()
        if AC_LIKE_RE.match(stripped) and not HEADER_RE.match(stripped):
            in_ac = True
            cur = (i, [])
            blocks.append(cur)
            continue
        if in_ac:
            m = ITEM_RE.match(ln)
            if m:
                item = m.group(1).strip()
                if not PLACEHOLDER_RE.match(item):
                    cur[1].append((i, item))
            elif re.match(r"^#{1,3}\s", ln) or stripped.startswith("**"):
                in_ac = False
                cur = None
    return blocks


def _probe_gate():
    """O gate vivo enxerga um AC malformado depois de uma linha em branco?"""
    items = collect_ac_items(PROBE_LINES)
    if not items:
        return False
    ok, _ = classify_ears(items[0][1])
    return not ok


def main(argv=None):
    p = argparse.ArgumentParser(
        prog="count_ears_visibility.py",
        description="Mede a cegueira do gate de EARS (ISSUE-050).",
    )
    p.add_argument("--root", default=".")
    p.add_argument(
        "--legacy",
        action="store_true",
        help="mede com a regra antiga (linha em branco encerra o bloco)",
    )
    args = p.parse_args(argv)

    features = os.path.join(args.root, ".specs", "features")
    if not os.path.isdir(features):
        print(
            "count_ears_visibility: .specs/features nao encontrado em " + args.root,
            file=sys.stderr,
        )
        return 2

    specs = 0
    total_seen = 0
    total_present = 0
    blind_blocks = []
    blind_specs = set()
    unrecognized = []
    unrecognized_total = 0

    for name in sorted(os.listdir(features)):
        spec_path = os.path.join(features, name, "spec.md")
        if not os.path.isfile(spec_path):
            continue
        specs += 1
        with open(spec_path, "r", encoding="utf-8") as f:
            lines = f.read().splitlines()

        present_blocks = _items_of(lines, ends_on_blank_line=False)
        present = [item for _, items in present_blocks for item in items]
        if args.legacy:
            seen = [
                item
                for _, items in _items_of(lines, ends_on_blank_line=True)
                for item in items
            ]
        else:
            seen = collect_ac_items(lines)
        seen_lines = {ln for ln, _ in seen}

        total_present += len(present)
        total_seen += len(seen)

        for header, items in present_blocks:
            missing = [ln for ln, _ in items if ln not in seen_lines]
            if missing:
                blind_blocks.append((name, header, missing))
                blind_specs.add(name)

        for header, items in _unrecognized_blocks(lines):
            if items:
                unrecognized.append((name, header, len(items)))
                unrecognized_total += len(items)

    mode = " (legacy)" if args.legacy else ""
    print(
        f"count_ears_visibility{mode}: specs={specs} | vistos={total_seen} "
        f"presentes={total_present} cegos={total_present - total_seen}"
    )
    print(
        f"blocos cegos: {len(blind_blocks)} | specs com bloco cego: {len(blind_specs)}"
    )
    for name, header, missing in blind_blocks:
        print(f"  {name}:{header} ({len(missing)} AC(s) cego(s))")
    if unrecognized:
        print(
            f"diagnostico (fora da ISSUE-050): {unrecognized_total} AC(s) sob cabecalho "
            f"que o validador nao reconhece, em {len(unrecognized)} bloco(s):"
        )
        for name, header, count in unrecognized:
            print(f"  {name}:{header} ({count} AC(s))")

    if not args.legacy:
        probe_ok = _probe_gate()
        print(
            "gate ve o AC de prova (AC sem SHALL depois de linha em branco): "
            + ("sim" if probe_ok else "NAO")
        )
        if not probe_ok:
            print("ERROR: o gate esta cego - o reset de linha em branco voltou.", file=sys.stderr)
            return 1

    return 1 if blind_blocks else 0


if __name__ == "__main__":
    raise SystemExit(main())
