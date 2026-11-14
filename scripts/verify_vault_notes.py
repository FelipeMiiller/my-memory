#!/usr/bin/env python3
"""Verifica as notas de conceito do Cofre Central (ADR-064).

A regra da ADR-064 e: *todo conceito que uma nota do cofre linka tem uma
nota*, com o nome do arquivo igual ao texto do wikilink. Este script verifica
o que a regra exige de cada nota de conceito:

  - encoding (UTF-8 sem BOM), finais de linha LF, nenhum caractere CJK
  - frontmatter canonico: title, category, summary, tags, status
  - `title` igual ao texto do wikilink que referencia a nota
  - H1 do corpo igual ao `title`
  - secoes "O que e", "Onde entra aqui" e "Relacionado"
  - backlink para o README da pasta e para a nota que a linka
  - todo wikilink do corpo aponta para um arquivo que EXISTE no vault

Uso:
    python scripts/verify_vault_notes.py [--vault <raiz>] [--repo <slug>]

A raiz do vault vem, nesta ordem: --vault, MY_MEMORY_CENTRAL_VAULT, o
`central_vault.path` do config global (~/.memory/config.yaml).

Exit: 0 se tudo passa, 1 se ha erros.
"""

import argparse
import os
import re
import sys

# (wikilink, pasta, nota de origem) — os 13 conceitos da ADR-064
NOTAS = [
    ("Docker", "cloud", "blue-green-deployment-obsidian"),
    ("Kubernetes", "cloud", "blue-green-deployment-obsidian"),
    ("Observabilidade", "cloud", "blue-green-deployment-obsidian"),
    ("Rollback", "cloud", "blue-green-deployment-obsidian"),
    ("Expand and Contract", "cloud", "blue-green-deployment-obsidian"),
    ("SRE e SLOs", "cloud", "blue-green-deployment-obsidian"),
    ("Banco de Dados", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Índices SQL", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Busca Vetorial e RAG", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Filas e Processamento Assíncrono", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Transações e Concorrência", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Observabilidade de Banco de Dados", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
    ("Backup e Recuperação", "database", "PostgreSQL — Visão Geral de Recursos e Ferramentas"),
]

CJK = re.compile(r"[\u4e00-\u9fff\uff00-\uffef]+")
WIKILINK = re.compile(r"\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]")
GLOBAL_CONFIG = os.path.join(os.path.expanduser("~"), ".memory", "config.yaml")


def resolver_raiz(vault_cli):
    if vault_cli:
        return vault_cli
    env = os.environ.get("MY_MEMORY_CENTRAL_VAULT")
    if env:
        return env
    if os.path.exists(GLOBAL_CONFIG):
        with open(GLOBAL_CONFIG, encoding="utf-8") as fh:
            for linha in fh:
                m = re.match(r"^\s*path:\s*(.+)$", linha)
                if m:
                    return os.path.expandvars(m.group(1).strip())
    return ""


def frontmatter(text):
    """Separa o frontmatter YAML simples. Devolve (campos, corpo)."""
    if not text.startswith("---"):
        return None, text
    fim = text.find("\n---", 3)
    if fim == -1:
        return None, text
    bruto = text[3:fim]
    corpo = text[fim + 4:]
    campos = {}
    chave = None
    for linha in bruto.split("\n"):
        if re.match(r"^[a-z_]+:", linha):
            chave, _, valor = linha.partition(":")
            chave = chave.strip()
            campos[chave] = valor.strip().strip('"')
        elif chave and linha.strip().startswith("-"):
            campos.setdefault(chave + "_lista", []).append(
                linha.strip()[1:].strip().strip('"')
            )
    return campos, corpo


def main():
    ap = argparse.ArgumentParser(description="Verifica as notas de conceito do cofre")
    ap.add_argument("--vault", default="", help="raiz do vault")
    args = ap.parse_args()

    vault = resolver_raiz(args.vault)
    if not vault or not os.path.isdir(vault):
        print(f"vault nao encontrado: {vault!r}")
        return 2

    erros = 0
    avisos = 0

    # indice de arquivos do vault (base sem extensao + caminho relativo), para
    # conferir que todo wikilink aponta para algo que existe
    existentes = set()
    for raiz, _, arquivos in os.walk(vault):
        for arq in arquivos:
            if not arq.endswith(".md"):
                continue
            existentes.add(os.path.splitext(arq)[0])
            relativos = os.path.relpath(os.path.join(raiz, arq), vault)[:-3]
            existentes.add(relativos.replace("\\", "/"))

    for titulo, pasta, origem in NOTAS:
        caminho = os.path.join(vault, pasta, titulo + ".md")
        problemas = []

        if not os.path.exists(caminho):
            print(f"[ERRO] {titulo}: arquivo ausente em {caminho}")
            erros += 1
            continue

        bruto = open(caminho, "rb").read()
        if bruto.startswith(b"\xef\xbb\xbf"):
            problemas.append("tem BOM")
        texto = bruto.decode("utf-8")
        if "\r\n" in texto:
            problemas.append("tem CRLF")

        cjk = CJK.findall(texto)
        if cjk:
            problemas.append("CJK espurio: " + " ".join(cjk))

        campos, corpo = frontmatter(texto)
        if campos is None:
            problemas.append("sem frontmatter")
            campos = {}

        for obrig in ("title", "category", "summary", "tags", "status"):
            if obrig not in campos:
                problemas.append(f"frontmatter sem '{obrig}'")
        if campos.get("title") != titulo:
            problemas.append(f"title '{campos.get('title')}' != '{titulo}'")
        if campos.get("category") != "resource":
            problemas.append(f"category '{campos.get('category')}' != 'resource'")
        if campos.get("status") != "ativo":
            problemas.append(f"status '{campos.get('status')}' != 'ativo'")
        if not campos.get("summary"):
            problemas.append("summary vazio")
        if not campos.get("tags_lista"):
            problemas.append("tags vazio")

        h1 = re.search(r"^# (.+)$", corpo, re.MULTILINE)
        if not h1:
            problemas.append("sem H1")
        elif h1.group(1).strip() != titulo:
            problemas.append(f"H1 '{h1.group(1).strip()}' != '{titulo}'")

        for secao in ("O que é", "Onde entra aqui", "Relacionado"):
            if f"## {secao}" not in corpo:
                problemas.append(f"sem secao '{secao}'")

        if f"[[{pasta}/README]]" not in corpo:
            problemas.append(f"sem backlink para [[{pasta}/README]]")
        if f"[[{origem}]]" not in corpo:
            problemas.append(f"sem backlink para [[{origem}]]")

        for alvo in WIKILINK.findall(corpo):
            limpo = alvo.strip()
            if limpo in existentes:
                continue
            if any(e.lower() == limpo.lower() for e in existentes):
                avisos += 1
                print(f"[AVISO] {titulo}: wikilink [[{limpo}]] casa por case com um arquivo")
                continue
            problemas.append(f"wikilink quebrado: [[{limpo}]]")

        if problemas:
            erros += len(problemas)
            print(f"[ERRO] {titulo} ({pasta}):")
            for p in problemas:
                print(f"        - {p}")
        else:
            print(f"[OK] {titulo} ({pasta})")

    print()
    print(f"notas verificadas: {len(NOTAS)} | erros: {erros} | avisos: {avisos}")
    return 1 if erros else 0


if __name__ == "__main__":
    sys.exit(main())
