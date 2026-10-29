#!/usr/bin/env python3
"""
test_validate_state.py - testes do gate deterministico validate_state.py.

Cobre a regra de decisao do veredito:
  - PASS com evidencia file:line
  - PASS sem evidencia (evidence-or-zero)
  - FAIL
  - template nao preenchido '[PASS | FAIL]'
  - nenhuma linha candidata -> 'sem veredito' (o corpo do relatorio NAO e palha)
  - PARTIAL numa linha que tambem contem 'PASS' -> partial, nunca pass
  - linhas 'Veredito:' / 'Resultado:' / 'Verdict:' reconhecidas
  - precedencia: primeiro candidato decisivo vence; PARTIAL tem precedencia

Padrao do repo: asserts + main() proprio (sem pytest).
"""

import os
import shutil
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from validate_state import _check_feature, _verdict  # noqa: E402


def _write_report(body, suffix=".md"):
    """Cria um feature dir temporario com o validation.md dado."""
    tmp = tempfile.mkdtemp(prefix="validate_state_test_")
    fdir = os.path.join(tmp, "feat")
    os.makedirs(fdir, exist_ok=True)
    with open(os.path.join(fdir, "validation" + suffix), "w", encoding="utf-8") as fh:
        fh.write(body)
    return tmp, fdir


def test_verdict_none_without_candidate():
    """Sem nenhuma linha candidata -> None (nao 'pass', nao 'unfilled')."""
    report = (
        "# Validation Report: feat\n\n"
        "O verificadorindependente rodou a suite inteira.\n"
        "Todos os 53 criterios foram satisfeitos e 2uvegas foram corrigidas.\n"
        "Nenhum campo de veredito existe neste arquivo.\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"


def test_verdict_prose_only_is_not_unfilled():
    """Relatorio em prosa com PASS e FAIL no corpo nao vira 'unfilled'."""
    report = (
        "# Relatorio de validacao\n\n"
        "A tabela traz 53 PASS e 2 FAIL no corpo do texto.\n"
        "O verificador fechou os gaps.\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"


def test_verdict_pass_canonical_block():
    """O bloco canonico '## Validation: PASS' + '**Result**: PASS' e PASS."""
    report = (
        "# Validation\n\n"
        "## Validation: PASS\n\n"
        "**Result**: PASS\n\n"
        "Evidencia: internal/embedder/builtin.go:62\n"
    )
    assert _verdict(report) == "pass", f"esperado 'pass', obtido {_verdict(report)!r}"


def test_verdict_pass_with_evidence():
    """PASS com evidencia file:line nao gera erro nenhum."""
    body = (
        "# Validation Report: feat\n\n"
        "**Verdict**: PASS\n\n"
        "Evidencia: internal/embedder/builtin.go:143\n"
    )
    tmp, fdir = _write_report(body)
    try:
        errors = _check_feature(fdir, "feat")
        assert errors == [], f"esperado sem erros, obtido {errors!r}"
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_verdict_pass_without_evidence():
    """PASS sem nenhuma citacao file:line viola evidence-or-zero."""
    body = (
        "# Validation Report: feat\n\n"
        "**Verdict**: PASS\n\n"
        "Tudo certo, confia.\n"
    )
    tmp, fdir = _write_report(body)
    try:
        errors = _check_feature(fdir, "feat")
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert "cites no file:line evidence" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_verdict_fail():
    """FAIL e reportado como FAIL."""
    body = (
        "# Validation Report: feat\n\n"
        "**Verdict**: FAIL\n\n"
        "Evidencia: internal/embedder/builtin.go:143\n"
    )
    tmp, fdir = _write_report(body)
    try:
        errors = _check_feature(fdir, "feat")
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert "verdict is FAIL" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_verdict_unfilled_template():
    """O placeholder '[PASS | FAIL]' continua sendo detectado."""
    body = (
        "# Validation Report: feat\n\n"
        "**Result**: [PASS | FAIL]\n\n"
        "Evidencia: internal/embedder/builtin.go:143\n"
    )
    tmp, fdir = _write_report(body)
    try:
        errors = _check_feature(fdir, "feat")
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert "template placeholder" in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_verdict_partial_is_not_pass():
    """PARTIAL numa linha que tambem contem 'PASS' -> partial, nunca pass."""
    line = (
        "**Verdict**: **PARTIAL** - PASS for 14 ACs, "
        "SPEC-GAP for 3 ACs (CA-01 / CA-12 / CA-16), 1 surviving mutant."
    )
    report = "# Validation: feat\n\n" + line + "\n"
    assert _verdict(report) == "partial", f"esperado 'partial', obtido {_verdict(report)!r}"

    body = report + "\nEvidencia: internal/codeast/ranking.go:13\n"
    tmp, fdir = _write_report(body)
    try:
        errors = _check_feature(fdir, "feat")
        assert len(errors) == 1, f"esperado 1 erro, obtido {errors!r}"
        assert "PARTIAL" in errors[0], errors[0]
        # evidence-or-zero nao vale para partial: so PASS exige evidencia.
        assert "cites no file:line evidence" not in errors[0], errors[0]
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def test_verdict_partial_has_precedence_over_later_pass():
    """PARTIAL em qualquer linha mantem o arquivo vermelho, mesmo com PASS depois."""
    report = (
        "# Validation: feat\n\n"
        "**Verdict**: **PARTIAL** - PASS for 14 ACs\n\n"
        "## Validation: PASS\n"
    )
    assert _verdict(report) == "partial", f"esperado 'partial', obtido {_verdict(report)!r}"


def test_verdict_first_decisive_candidate_wins():
    """Sem PARTIAL, o primeiro candidato decisivo vence."""
    report = (
        "# Validation: feat\n\n"
        "**Verdict**: FAIL\n\n"
        "## Validation: PASS\n"
    )
    assert _verdict(report) == "fail", f"esperado 'fail', obtido {_verdict(report)!r}"


def test_verdict_labeled_lines_recognized():
    """'Veredito:', 'Resultado:' e 'Verdict:' sao linhas candidatas."""
    assert _verdict("# X\n\n**Veredito**: PASS\n") == "pass"
    assert _verdict("# X\n\n**Resultado**: PASS\n") == "pass"
    assert _verdict("# X\n\n**Verdict**: PASS\n") == "pass"
    assert _verdict("# X\n\n**Result**: PASS\n") == "pass"
    assert _verdict("# X\n\nveredito: PASS\n") == "pass"
    assert _verdict("# X\n\nresultado: PASS\n") == "pass"
    # Negrito opcional, qualquer caixa.
    assert _verdict("# X\n\n**VEREDITO**: PASS\n") == "pass"
    assert _verdict("# X\n\nResultado: PASS\n") == "pass"


def test_verdict_count_line_is_not_a_verdict():
    """Uma linha de contagem ('- **PARTIAL / SPEC-GAP**: 0') nomeia um campo, nao declara um veredito."""
    report = (
        "# Validation: feat\n\n"
        "### Counts\n"
        "- **PASS**: 16 / 16 ACs\n"
        "- **PARTIAL / SPEC-GAP**: 0\n"
        "- **FAIL**: 0\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"

    # Com o bloco canonico, o veredito continua PASS - a contagem nao o altera.
    report_ok = report + "\n## Validation: PASS\n\n**Result**: PASS\n"
    assert _verdict(report_ok) == "pass", f"esperado 'pass', obtido {_verdict(report_ok)!r}"


def test_verdict_quoted_label_is_not_a_declaration():
    """Um rotulo citado em code span e um EXEMPLO de rotulo, nao uma declaracao."""
    report = (
        "# Validation: feat\n\n"
        "O trecho abaixo (`**Verdict**: **PARTIAL**`) explica o que seria um PARTIAL.\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"


def test_verdict_leading_token_must_be_emphasized():
    """O token que abre a linha precisa ser enfase para ser uma declaracao."""
    report = (
        "# Validation: feat\n\n"
        "PARTIAL e um veredito ao lado de PASS e FAIL.\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"

    # Com enfase, e uma declaracao.
    report_bold = "# Validation: feat\n\n**PARTIAL** - 3 ACs em SPEC-GAP.\n"
    assert _verdict(report_bold) == "partial", f"esperado 'partial', obtido {_verdict(report_bold)!r}"


def test_verdict_leading_token_line():
    """Uma linha que ABRE com o token de veredito tambem e candidata."""
    report = "# Validacao\n\n**PASS** - 25/25 criterios satisfeitos, com prova.\n"
    assert _verdict(report) == "pass", f"esperado 'pass', obtido {_verdict(report)!r}"
    report_fail = "# Validacao\n\n**FAIL** - 3 gaps abertos.\n"
    assert _verdict(report_fail) == "fail", f"esperado 'fail', obtido {_verdict(report_fail)!r}"


def test_verdict_prose_resultado_line_is_not_decisive():
    """Uma linha de prosa 'Resultado:' sem palavra de veredito nao decide nada."""
    report = (
        "# Validation: feat\n\n"
        "**Resultado**: 19 packages, **0 falhas**.\n\n"
        "Outro caso:\n"
        "*Resultado:* Falha imediata em TestX. Mutante eliminado.\n"
    )
    assert _verdict(report) is None, f"esperado None, obtido {_verdict(report)!r}"


def test_verdict_pass_with_gaps_is_not_unfilled():
    """'PASS-WITH-GAPS' com PASS e FAIL na mesma linha nao e placeholder."""
    report = (
        "# Validation: feat\n\n"
        "**Verdict**: **PASS-WITH-GAPS** - 47 PASS - 3 FAIL - 2 UNVERIFIABLE (52 ACs)\n"
    )
    assert _verdict(report) == "pass", f"esperado 'pass', obtido {_verdict(report)!r}"


def main():
    print("\n==================================================")
    print("  Iniciando suite de testes do validate_state.py")
    print("==================================================\n")
    test_verdict_none_without_candidate()
    print("[OK] Sem linha candidata -> sem veredito (o corpo nao e palha).")
    test_verdict_prose_only_is_not_unfilled()
    print("[OK] Relatorio em prosa nao vira 'unfilled'.")
    test_verdict_pass_canonical_block()
    print("[OK] Bloco canonico '## Validation: PASS' reconhecido.")
    test_verdict_pass_with_evidence()
    print("[OK] PASS com evidencia file:line -> sem erro.")
    test_verdict_pass_without_evidence()
    print("[OK] PASS sem evidencia -> evidence-or-zero violated.")
    test_verdict_fail()
    print("[OK] FAIL reportado como FAIL.")
    test_verdict_unfilled_template()
    print("[OK] Template '[PASS | FAIL]' detectado.")
    test_verdict_partial_is_not_pass()
    print("[OK] PARTIAL com 'PASS' na mesma linha -> partial (nunca pass).")
    test_verdict_partial_has_precedence_over_later_pass()
    print("[OK] PARTIAL tem precedencia sobre um PASS posterior.")
    test_verdict_first_decisive_candidate_wins()
    print("[OK] Primeiro candidato decisivo vence.")
    test_verdict_labeled_lines_recognized()
    print("[OK] Linhas Veredito:/Resultado:/Verdict:/Result: reconhecidas.")
    test_verdict_leading_token_line()
    print("[OK] Linha que abre com o token de veredito e candidata.")
    test_verdict_prose_resultado_line_is_not_decisive()
    print("[OK] Linha de prosa 'Resultado:' nao decide o veredito.")
    test_verdict_count_line_is_not_a_verdict()
    print("[OK] Linha de contagem ('- **PARTIAL / SPEC-GAP**: 0') nao e veredito.")
    test_verdict_quoted_label_is_not_a_declaration()
    print("[OK] Rotulo citado em code span nao e declaracao.")
    test_verdict_leading_token_must_be_emphasized()
    print("[OK] Token que abre a linha precisa ser enfase.")
    test_verdict_pass_with_gaps_is_not_unfilled()
    print("[OK] 'PASS-WITH-GAPS' nao e confundido com placeholder.")
    print("\n[OK] Todos os testes do validate_state.py passaram com 100% de sucesso!\n")


if __name__ == "__main__":
    main()
