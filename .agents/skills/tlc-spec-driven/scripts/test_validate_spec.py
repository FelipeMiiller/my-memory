#!/usr/bin/env python3
"""
test_validate_spec.py - testes do gate deterministico validate_spec.py.

Cobre a coleta dos Acceptance Criteria (a cegueira da ISSUE-050):
  - o bloco com linha em branco depois do cabecalho e o bloco sem ela
    (os dois formatos sao validos e produzem o mesmo resultado)
  - AC sem SHALL -> ERROR nomeando a linha e o criterio
  - AC com SHALL e sem keyword EARS -> WARNING
  - placeholder de template -> sem erro
  - o bloco termina em heading (#{1,3}) ou em linha iniciada por **
  - lista numerada fora de um bloco de AC -> nao julgada
  - o reset de linha em branco nao volta (a suite falha se ele voltar)

Padrao do repo: asserts + main() proprio (sem pytest).
"""

import inspect
import os
import shutil
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import validate_spec  # noqa: E402
from validate_spec import check, collect_ac_items  # noqa: E402


def _fixture(ac_lines, after="", before_ac=""):
    """Spec minima (com as secoes exigidas pelo gate) e o bloco de AC dado.

    ac_lines    linhas do bloco de AC. A primeira linha vazia reproduz o
                formato com linha em branco depois do cabecalho.
    after       conteudo depois do bloco de AC.
    before_ac   conteudo antes do bloco de AC.
    """
    body = (
        "# Feature: fixture\n\n"
        "## Problem Statement\n\n"
        "Problema do fixture.\n\n"
        + before_ac
        + "\n## Out of Scope\n\n"
        "Fora do escopo.\n\n"
        "## Assumptions & Open Questions\n\n"
        "| Assumption | Chosen default | Rationale |\n"
        "| --- | --- | --- |\n"
        "| A choice | B default | C reason |\n\n"
        "Open questions: none\n\n"
        "## User Stories\n\n"
        "### P1: historia\n\n"
        "**Acceptance Criteria**:\n"
    )
    for ln in ac_lines:
        body += ln + "\n"
    body += after
    body += (
        "\n## Requirement Traceability\n\n"
        "| Requirement ID | Acceptance Criterion | Where | Status |\n"
        "| --- | --- | --- | --- |\n"
        "| FIX-01 | AC1 | x | Verified |\n"
    )
    return body


def _write_spec(body):
    """Escreve o fixture num spec.md temporario e devolve (dir, path)."""
    tmp = tempfile.mkdtemp(prefix="validate_spec_test_")
    path = os.path.join(tmp, "spec.md")
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(body)
    return tmp, path


def _line_of(path, needle):
    """Numero (1-based) da linha que contem o trecho dado."""
    with open(path, "r", encoding="utf-8") as fh:
        for i, ln in enumerate(fh.read().splitlines(), start=1):
            if needle in ln:
                return i
    return -1


def _criterion_of(message):
    """Texto do criterio extraido da mensagem de erro do gate."""
    return message.split(": ", 2)[-1]


def test_ac_without_shall_after_blank_line_is_error():
    """AC sem SHALL depois de uma linha em branco e ERROR (EAG-01, EAG-03)."""
    body = _fixture(["", "1. O sistema cadastra o usuario sem confirmacao."])
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
        line = _line_of(path, "O sistema cadastra o usuario")
        assert errors[0].startswith(f"L{line}:"), errors[0]
        assert "no SHALL" in errors[0], errors[0]
        assert "cadastra o usuario" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_ac_without_shall_without_blank_line_is_error():
    """O mesmo AC sem linha em branco e ERROR tambem (EAG-06)."""
    body = _fixture(["1. O sistema cadastra o usuario sem confirmacao."])
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
        assert "no SHALL" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_both_block_formats_produce_the_same_errors():
    """Com e sem linha em branco, o gate produz o mesmo numero de erros (EAG-06)."""
    tmp_a, path_a = _write_spec(
        _fixture(["", "1. O sistema cadastra o usuario sem confirmacao."])
    )
    tmp_b, path_b = _write_spec(
        _fixture(["1. O sistema cadastra o usuario sem confirmacao."])
    )
    try:
        errors_a, _ = check(path_a)
        errors_b, _ = check(path_b)
        assert len(errors_a) == len(errors_b) == 1, (errors_a, errors_b)
        assert _criterion_of(errors_a[0]) == _criterion_of(errors_b[0]), (
            errors_a,
            errors_b,
        )
    finally:
        shutil.rmtree(tmp_a, ignore_errors=True)
        shutil.rmtree(tmp_b, ignore_errors=True)


def test_block_ends_at_a_heading():
    """O bloco termina em heading: a lista seguinte nao e julgada (EAG-02)."""
    body = _fixture(
        ["", "1. O sistema cadastra o usuario sem confirmacao."],
        after="\n## Outra secao\n\n1. Uma lista numerada qualquer, fora do bloco de AC.\n",
    )
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
        assert "cadastra o usuario" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_block_ends_at_a_bold_line():
    """O bloco termina em linha iniciada por **: a lista seguinte nao e julgada (EAG-02)."""
    body = _fixture(
        ["", "1. O sistema cadastra o usuario sem confirmacao."],
        after="\n**Nota**: a lista abaixo nao e um AC.\n\n1. Uma lista numerada qualquer, fora do bloco de AC.\n",
    )
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
        assert "cadastra o usuario" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_ac_with_shall_but_no_ears_keyword_is_warning():
    """AC com SHALL e sem keyword EARS e WARNING, nao ERROR (EAG-04)."""
    body = _fixture(["1. Cada criterio malformado SHALL ser reportado pelo gate."])
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert errors == [], f"esperado sem erros, obtido {errors!r}"
        assert len(warnings) == 1, f"esperado 1 warning, obtido {warnings!r}"
        assert "no EARS keyword" in warnings[0], warnings[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_placeholder_criterion_is_not_an_error():
    """Placeholder de template nao vira erro (EAG-05)."""
    body = _fixture(["1. [Something to be decided]"])
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert errors == [], f"esperado sem erros, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_numbered_list_outside_an_ac_block_is_not_judged():
    """Lista numerada fora de um bloco de AC nao e julgada (EAG-02)."""
    body = _fixture(
        ["1. WHEN x THEN y SHALL z."],
        before_ac="1. Uma lista numerada qualquer, fora de bloco de AC.\n",
    )
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert errors == [], f"esperado sem erros, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_well_formed_spec_has_no_errors_or_warnings():
    """Uma spec bem formada passa limpa: o gate nao cria erro por conta propria."""
    body = _fixture(["1. WHEN x THEN y SHALL z."])
    tmp, path = _write_spec(body)
    try:
        errors, warnings = check(path)
        assert errors == [], f"esperado sem erros, obtido {errors!r}"
        assert warnings == [], f"esperado sem warnings, obtido {warnings!r}"
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_collect_ac_items_keeps_criteria_after_a_blank_line():
    """O coletor segue coletando depois da linha em branco (EAG-01)."""
    lines = [
        "**Acceptance Criteria**:",
        "",
        "1. WHEN x THEN y SHALL z.",
        "2. WHEN a THEN b SHALL c.",
    ]
    items = collect_ac_items(lines)
    assert len(items) == 2, f"esperado 2 criterios, obtido {items!r}"
    assert items[0][1].startswith("WHEN x"), items
    assert items[1][1].startswith("WHEN a"), items


def test_collect_ac_items_ignores_lists_outside_a_block():
    """O coletor so coleta dentro de um bloco declarado (EAG-02)."""
    lines = [
        "1. Uma lista qualquer.",
        "## Secao",
        "**Acceptance Criteria**:",
        "1. WHEN x THEN y SHALL z.",
    ]
    items = collect_ac_items(lines)
    assert len(items) == 1, f"esperado 1 criterio, obtido {items!r}"
    assert items[0][1].startswith("WHEN x"), items


def test_collect_ac_items_skips_placeholders():
    """Placeholder de template nao entra na coleta (EAG-05)."""
    lines = [
        "**Acceptance Criteria**:",
        "1. [Something to be decided]",
        "2. WHEN x THEN y SHALL z.",
    ]
    items = collect_ac_items(lines)
    assert len(items) == 1, f"esperado 1 criterio, obtido {items!r}"
    assert items[0][1].startswith("WHEN x"), items


def test_collect_ac_items_accepts_header_variants():
    """O coletor aceita cabecalhos com colon dentro do bold e na forma de heading (ISSUE-051)."""
    lines_colon_inside = [
        "**Acceptance Criteria:**",
        "1. WHEN event THEN system SHALL act.",
    ]
    items1 = collect_ac_items(lines_colon_inside)
    assert len(items1) == 1, f"esperado 1 criterio com colon inside, obtido {items1!r}"

    lines_h2 = [
        "## Acceptance Criteria",
        "1. WHEN event THEN system SHALL act.",
    ]
    items2 = collect_ac_items(lines_h2)
    assert len(items2) == 1, f"esperado 1 criterio com h2 heading, obtido {items2!r}"

    lines_h3 = [
        "### Acceptance Criteria:",
        "1. WHEN event THEN system SHALL act.",
    ]
    items3 = collect_ac_items(lines_h3)
    assert len(items3) == 1, f"esperado 1 criterio com h3 heading, obtido {items3!r}"


def test_collector_source_has_no_blank_line_reset():
    """O reset de linha em branco nao pode voltar ao coletor (EAG-09)."""
    src = inspect.getsource(validate_spec.collect_ac_items)
    assert 'stripped == ""' not in src, "o reset de linha em branco voltou ao coletor"


def main():
    print("\n==================================================")
    print("  Iniciando suite de testes do validate_spec.py")
    print("==================================================\n")
    test_ac_without_shall_after_blank_line_is_error()
    print("[OK] AC sem SHALL depois de linha em branco -> ERROR com linha e criterio.")
    test_ac_without_shall_without_blank_line_is_error()
    print("[OK] AC sem SHALL sem linha em branco -> ERROR.")
    test_both_block_formats_produce_the_same_errors()
    print("[OK] Os dois formatos de bloco produzem o mesmo resultado.")
    test_block_ends_at_a_heading()
    print("[OK] O bloco termina em heading.")
    test_block_ends_at_a_bold_line()
    print("[OK] O bloco termina em linha iniciada por **.")
    test_ac_with_shall_but_no_ears_keyword_is_warning()
    print("[OK] AC com SHALL e sem keyword -> WARNING.")
    test_placeholder_criterion_is_not_an_error()
    print("[OK] Placeholder de template -> sem erro.")
    test_numbered_list_outside_an_ac_block_is_not_judged()
    print("[OK] Lista numerada fora de bloco de AC -> nao julgada.")
    test_well_formed_spec_has_no_errors_or_warnings()
    print("[OK] Spec bem formada -> sem erro e sem warning.")
    test_collect_ac_items_keeps_criteria_after_a_blank_line()
    print("[OK] O coletor segue depois da linha em branco.")
    test_collect_ac_items_ignores_lists_outside_a_block()
    print("[OK] O coletor so coleta dentro de um bloco declarado.")
    test_collect_ac_items_skips_placeholders()
    print("[OK] O coletor ignora placeholders.")
    test_collect_ac_items_accepts_header_variants()
    print("[OK] O coletor aceita variantes de cabecalho (colon inside, heading).")
    test_collector_source_has_no_blank_line_reset()
    print("[OK] O reset de linha em branco nao esta no coletor.")
    print("\n[OK] Todos os testes do validate_spec.py passaram com 100% de sucesso!\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
