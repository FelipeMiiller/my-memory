import { describe, expect, it } from "vitest"
import {
  extractWikilinks,
  parseWikilink,
  preprocessWikilinks,
} from "./wikilinks"

/**
 * Tabela de casos do pré-processador de wikilinks.
 *
 * Este é o **espelho** da tabela do parser Go (`internal/parser/wikilinks.go`).
 * Se os dois divergirem, a viewer passa a mostrar links que o grafo não tem —
 * ou seja, a mentir sobre a topologia. Por isso o caso a caso importa.
 */
describe("pré-processador de wikilinks", () => {
  const cases: Array<{
    nome: string
    entrada: string
    esperado: string
  }> = [
    {
      nome: "link simples",
      entrada: "Veja [[Arquitetura]] agora.",
      esperado: "Veja [Arquitetura](/notes?path=Arquitetura) agora.",
    },
    {
      nome: "link com alias",
      entrada: "Veja [[Arquitetura|O design]] agora.",
      esperado: "Veja [O design](/notes?path=Arquitetura) agora.",
    },
    {
      nome: "link com âncora",
      entrada: "Veja [[Arquitetura#Visão Geral]].",
      esperado:
        "Veja [Arquitetura](/notes?path=Arquitetura&anchor=Vis%C3%A3o%20Geral).",
    },
    {
      nome: "link com âncora e alias",
      entrada: "Veja [[Arquitetura#Visão Geral|o design]].",
      esperado:
        "Veja [o design](/notes?path=Arquitetura&anchor=Vis%C3%A3o%20Geral).",
    },
    {
      nome: "link com block id",
      entrada: "Veja [[Arquitetura#^c182]].",
      esperado: "Veja [Arquitetura](/notes?path=Arquitetura&block=c182).",
    },
    {
      nome: "link interno do documento",
      entrada: "Veja [[#Conclusão]].",
      esperado: "Veja [Conclusão](#Conclus%C3%A3o).",
    },
    {
      nome: "embed",
      entrada: "Imagem: ![[diagrama]].",
      esperado: "Imagem: [diagrama](/notes?path=diagrama).",
    },
    {
      nome: "vault federado é texto literal",
      entrada: "Veja [[memory://central/standards/auth]].",
      esperado: "Veja [[memory://central/standards/auth]].",
    },
    {
      nome: "relação no alias não aparece no rótulo",
      entrada: "Veja [[Arquitetura|rel:implements]].",
      esperado: 'Veja [Arquitetura](/notes?path=Arquitetura "implements").',
    },
    {
      nome: "bloco de código cercado não vira link",
      entrada: "```\n[[Arquitetura]]\n```",
      esperado: "```\n[[Arquitetura]]\n```",
    },
    {
      nome: "código inline não vira link",
      entrada: "Use `[[Arquitetura]]` aqui.",
      esperado: "Use `[[Arquitetura]]` aqui.",
    },
    {
      nome: "comentário HTML não vira link",
      entrada: "<!-- [[Arquitetura]] -->",
      esperado: "<!-- [[Arquitetura]] -->",
    },
    {
      nome: "linha de tabela não vira link",
      entrada: "| nota | alvo |\n| --- | --- |\n| a | [[Arquitetura]] |",
      esperado: "| nota | alvo |\n| --- | --- |\n| a | [[Arquitetura]] |",
    },
  ]

  for (const testCase of cases) {
    it(testCase.nome, () => {
      expect(preprocessWikilinks(testCase.entrada)).toBe(testCase.esperado)
    })
  }

  it("não cria link para alvo inexistente ([[...]])", () => {
    expect(preprocessWikilinks("Veja [[...]].")).toBe("Veja [[...]].")
  })

  it("não cria link para o próprio diretório ([[.]])", () => {
    expect(preprocessWikilinks("Veja [[.]].")).toBe("Veja [[.]].")
  })

  it("preserva o frontmatter intacto", () => {
    const entrada = [
      "---",
      'title: "Nota"',
      'aliases: ["[[Arquitetura]]"]',
      "---",
      "",
      "Corpo com [[Arquitetura]].",
    ].join("\n")

    const saida = preprocessWikilinks(entrada)

    expect(saida).toContain('aliases: ["[[Arquitetura]]"]')
    expect(saida).toContain("Corpo com [Arquitetura](/notes?path=Arquitetura).")
  })

  it("trata vários wikilinks no mesmo parágrafo", () => {
    expect(preprocessWikilinks("[[A]] e [[B]]")).toBe(
      "[A](/notes?path=A) e [B](/notes?path=B)"
    )
  })
})

describe("parseWikilink", () => {
  it("extrai destino, âncora e alias", () => {
    const link = parseWikilink("Arquitetura#Visão Geral|O design", false)

    expect(link.target).toBe("Arquitetura")
    expect(link.anchor).toBe("Visão Geral")
    expect(link.alias).toBe("O design")
    expect(link.relation).toBe("links_to")
    expect(link.isSameDoc).toBe(false)
    expect(link.isFederated).toBe(false)
  })

  it("marca link interno do documento", () => {
    const link = parseWikilink("#Conclusão", false)

    expect(link.isSameDoc).toBe(true)
    expect(link.target).toBe("")
    expect(link.anchor).toBe("Conclusão")
  })

  it("separa o block id da âncora", () => {
    const link = parseWikilink("Arquitetura#^c182", false)

    expect(link.blockId).toBe("c182")
    expect(link.anchor).toBe("")
  })

  it("detecta relação pelo prefixo rel:", () => {
    const link = parseWikilink("rel:implements:Arquitetura", false)

    expect(link.relation).toBe("implements")
    expect(link.target).toBe("Arquitetura")
  })

  it("detecta relação por relação conhecida no prefixo", () => {
    const link = parseWikilink("depends_on:Arquitetura", false)

    expect(link.relation).toBe("depends_on")
    expect(link.target).toBe("Arquitetura")
  })

  it("detecta relação pelo alias", () => {
    const link = parseWikilink("Arquitetura|rel:implements", false)

    expect(link.relation).toBe("implements")
    expect(link.alias).toBe("rel:implements")
  })

  it("decompoe URI federada", () => {
    const link = parseWikilink("memory://central/standards/auth", false)

    expect(link.isFederated).toBe(true)
    expect(link.federatedRepo).toBe("central")
    expect(link.federatedPath).toBe("standards/auth")
    expect(link.target).toBe("memory://central/standards/auth")
  })

  it("não confunde memory:// com relação", () => {
    const link = parseWikilink("memory://central/standards/auth", false)

    expect(link.relation).toBe("links_to")
  })

  it("marca embed", () => {
    expect(parseWikilink("diagrama", true).isEmbed).toBe(true)
  })
})

describe("extractWikilinks", () => {
  it("ignora os contextos sem link, como o Go", () => {
    const texto = [
      "[[A]]",
      "```",
      "[[B]]",
      "```",
      "`[[C]]`",
      "<!-- [[D]] -->",
      "| [[E]] |",
    ].join("\n")

    const links = extractWikilinks(texto)

    expect(links.map((link) => link.target)).toEqual(["A"])
  })
})
