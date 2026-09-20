# ADR-046: Padrão "Deferred Until" para Fallbacks Opcionais em ADRs

- **Date**: 2026-09-19
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: methodology, architecture, yagni, adr-038, deferral, scope-reduction, cross-cutting

## Context and Problem Statement

ADRs Go do my-memory têm repetido o mesmo anti-pattern: propõem um **caminho primário** + um **"fallback opcional"** descrito em prosa vaga ("também pode ser usado se o usuário preferir", "caso precise", "está disponível como alternativa"). Exemplos recentes onde isso apareceu:

- **ADR-045** (ASR streaming) originalmente listava "worker Python sidecar (Whisper.cpp/Vosk/sherpa-onnx)" como "opcionalmente disponível como fallback secundário" — sem critério nenhum de promoção.
- **ADR-042** (mymemoryd com workers sidecar) trata Python sidecars como contrato permanente pra ASR/TTS/LLM, sem diferenciar quais estão ativos hoje vs. quais são reservas.
- **ADR-035** (embedder builtin) menciona "Fallback ONNX MiniLM" sem definir quando o fallback dispara além de "se falhar".
- Frameworks CLI candidatos (`--engine=X|Y|Z`, `--provider=onnx|python|onnxruntime`) tendem a aceitar múltiplos engines sem implementar todos desde o MVP.

Esse anti-pattern é dívida arquitetural por três motivos concretos:

1. **"Opcional" vira TODO silencioso.** Um dev que vê a frase "também disponível como fallback" interpreta como "implementar quando der tempo". O fallback vira código não-rastreado, sem testes, sem SLO.
2. **Aumenta surface area de manutenção sem benefício real.** Uma binding não testada quebra silenciosamente; um provider "opcional" duplica o caminho de configuração (`config.yaml`, schemas Go, help text, CLI flags, docs) sem que ninguém valide se o caminho é exercitado em produção.
3. **Vira ADR paralelo não-declarado.** Quando alguém implementa o fallback sem rastrear, está criando um ADR de fato sem o rito de um ADR — sem alternativas avaliadas, sem decisão documentada, sem critério de aceitação.

A sessão de 2026-09-19 sobre ADR-045 (Nemotron 3.5 ASR in-process) explicitou a escolha: ao invés de escrever "Python sidecar opcionalmente disponível como fallback secundário", reescrever como `## Deferral: Plano B (Python Sidecar)` com gatilhos verificáveis, lista do que fica proibido até promoção, procedimento de promoção e status atual. O Felipe pediu pra aplicar (`faça`) e o resultado foi mais útil — agora há um ponto-de-decisão rastreável pra quando (e se) a binding `yalue/onnxruntime_go` precisar de Plano B.

A pergunta que este ADR responde: **como padronizar o tratamento de "fallback opcional" em todos os ADRs do my-memory, de modo que a decisão seja rastreável, reversível e ganhe evidência antes de ser ativada?**

## Decision Drivers

- **YAGNI explícito** — só implementar o que tem caso de uso real hoje. Fallbacks sem demanda verificada ficam adiados até a demanda aparecer com evidência.
- **Rastreabilidade** — toda decisão de implementar um fallback deve ter um ADR ou issue aberto referenciando o gatilho que disparou. Sem "alguém implementou em silêncio".
- **Reversibilidade barata** — manter o caminho adiado custa zero; quando o gatilho dispara, o ADR já tem o template pronto (4 componentes fixos).
- **Consistência cross-ADR** — todos os ADRs que precisarem diferir um Plano B usam a mesma estrutura (`## Deferral: Plano B`). Leitura fica uniforme.
- **Cross-link explícito com ADR-038** — ADR-038 (Viewer com Site Único e Dataset Fixo) já pratica "scope reduction + deferred" como decisão de produto; este ADR eleva o mesmo princípio pra metodologia transversal.
- **Compatibilidade com `tlc-spec-driven`** — quando a promoção do deferred acontece, vira spec (`spec.md` → `tasks.md` → execução por batches com quality gate). Nenhum caminho fora do rito.

## Considered Options

1. **Manter "opcional" como status quo** — descartado: anti-pattern comprovado na sessão ADR-045; custa mais em manutenção do que economiza em optionalidade.
2. **Criar flag `optional: true` no frontmatter do ADR** — descartado: flag binária não captura gatilhos, proibições ou procedimento. Vira label sem enforcement.
3. **Documentar metodologia em `docs/ADR_GUIDE.md` (texto separado)** — descartado: o lugar canônico de metodologia é um ADR (mesma autoridade de "Accepted"). Texto separado tem enforcement zero.
4. **Seção `## Deferral: <Plano B>` como cláusula vinculante em qualquer ADR que adiar** — *Opção Escolhida*. Codifica o framework no mesmo nível de autoridade que o resto do ADR; 4 subseções obrigatórias.

## Decision Outcome

Adota-se a **Opção 4**: qualquer ADR do my-memory que propuser um **caminho primário + Plano B opcional** DEVE substituir a prosa vaga por uma seção `## Deferral: <Plano B>` com 4 subseções obrigatórias:

### 1. Condições-gatilho para promoção

Lista enumerada de eventos verificáveis que, **com evidência documental anexada** (issue, ADR de reversão, ou RFC datada e linkada), promovem o Plano B de `Deferred` pra `Promoted`. Cada gatilho precisa ser:

- **Específico** — descreve o evento, não a categoria ("binding X regredir" sim; "outra binding aparecer" não).
- **Verificável** — pode ser confirmado por log, issue de campo, métrica ou auditoria externa.
- **Datado** — quando aplicável, inclui janela temporal (ex: "≤90 dias após release upstream").

**Não é gatilho válido:** "Vai que precisa" / "Por via das dúvidas" / especulação sobre plataformas sem dados reais. Cada gatilho precisa de pelo menos um caso real documentado, não inferido.

### 2. O que fica proibido até a promoção

Lista enumerada de itens (arquivos, blocos de config, flags CLI, dependências, seções de doc) que **NÃO podem ser criados** enquanto o Plano B estiver deferred. O propósito é:

- Impedir implementação silenciosa.
- Manter surface area de manutenção enxuta.
- Forçar que cada adição seja precedida por promoção formal da seção.

A lista é específica ao contexto do ADR (não é boilerplate genérico). Exemplo pra ADR-045:

- ❌ Criar `internal/asr/python_sidecar.go` (qualquer arquivo com esse nome em qualquer pacote).
- ❌ Adicionar bloco `asr.python_sidecar` em `config.yaml` schemas ou no template.
- ❌ Documentar `--asr-provider=python-sidecar` em qualquer doc pública ou help text.

### 3. Como promover (quando um gatilho dispara)

Passos numerados que transformam o Plano B em código vivo. Tipicamente:

1. Abrir issue rastreável referenciando o ADR e o gatilho que disparou.
2. Criar ADR novo (slot reservado pelo ADR-pai, ex: ADR-047 quando ADR-045 promover) com análise do provider específico, estimativa de esforço, trade-offs de licença/tamanho.
3. Atualizar o ADR-pai movendo a subseção "Status" de **Deferred** para **Promoted** com link cruzado pro ADR novo.
4. Implementar via spec `tlc-spec-driven` (`spec.md` → `tasks.md` → execução em batches com quality gate).
5. Atualizar `docs/REFERENCES.md` e docs correlatas.

### 4. Status atual

Subseção final com:

- **Última revisão**: data.
- **Gatilhos disparados**: contador (0 se nenhum).
- **Evidence log**: descrição ou link pra evidência que disparou (vazio quando contador = 0).
- **Próxima revisão**: evento ou data que justifica revisitar a seção.

## First Application

A primeira aplicação desta metodologia foi consolidada retroativamente em **ADR-045** (commit `64711c7`, 2026-09-19). O Felipe pediu pra substituir "opcionalmente disponível como fallback secundário" pelo framework `## Deferral: Plano B (Python Sidecar)`. As 4 subseções foram instanciadas com:

1. **3 gatilhos**: incidente de produção com binding Go, modelo upstream sem ONNX em ≤90 dias, restrição contratual de cadeia de suprimentos.
2. **5 proibições**: arquivo `python_sidecar.go`, bloco `asr.python_sidecar` em config, flags CLI `--asr-provider=python-sidecar`, deps Python em `pyproject.toml`, promoção do ADR-042 sem passar por ADR novo.
3. **5 passos de promoção**: issue → ADR novo → update ADR-045 → spec tlc-spec-driven → references.md.
4. **Status**: 0 gatilhos, evidence log vazio, próxima revisão alinhada com F2.

A primeira aplicação validou que o framework funciona: o ADR-045 pós-edição tem **ponto-de-decisão rastreável** pra quando (e se) a binding `yalue/onnxruntime_go` precisar de Plano B, sem código pago à toa enquanto não precisar.

## Application Template (copy/paste para futuros ADRs)

```markdown
## Deferral: <Nome do Plano B>

<Frase de 1-2 linhas justificando o diferimento sob ADR-038 / YAGNI.>

Esta cláusula é parte vinculante deste ADR — promover <arquivo/bloco principal> sem cumprir os gatilhos abaixo é uma violação arquitetural e exige ADR próprio.

### Condições-gatilho para promoção

A seção sai do estado deferred **somente** quando **uma** das condições abaixo for atendida com evidência documental anexada:

1. **<Gatilho 1>** — <descrição específica, verificável, datada>.
2. **<Gatilho 2>** — <descrição>.
3. **<Gatilho 3>** — <descrição>.

**Não é gatilho válido:** <lista de pretextos que NÃO disparam>.

### O que fica proibido até a promoção

- ❌ <item 1: arquivo, bloco de config, flag, dep>.
- ❌ <item 2>.
- ❌ <item 3>.

### Como promover (quando um gatilho dispara)

1. <Passo 1: issue rastreável>.
2. <Passo 2: ADR novo (slot N+1) com análise>.
3. <Passo 3: atualizar este ADR movendo Status pra Promoted>.
4. <Passo 4: spec tlc-spec-driven>.
5. <Passo 5: atualizar docs/REFERENCES.md>.

### Status atual

- **Última revisão**: YYYY-MM-DD
- **Gatilhos disparados**: 0
- **Evidence log**: nenhum ainda
- **Próxima revisão**: <evento/data>
```

## Consequences

### Positive

- **Decisões rastreáveis** — todo Plano B tem issue/ADR de origem quando promovido. Sem "implementação silenciosa".
- **Surface area enxuta** — só existe código de Plano B se tem caso de uso real com evidência.
- **Reversibilidade barata** — promover é seguir 5 passos numerados. Não reescrever o ADR-pai do zero.
- **Consistência cross-ADR** — qualquer leitura de ADR novo bate na mesma estrutura `## Deferral`. Leitura fica previsível.
- **Compatibilidade com rito `tlc-spec-driven`** — promoção vira spec normal, sem atalho.
- **Enforcement pela estrutura** — `## Deferral` é seção nomeada; ausência dela num ADR que tenha Plano B é sinal de revisão pendente.

### Negative

- **Custo de escrever 4 subseções por Plano B** — ~30 linhas extras no ADR. Mitigação: copy/paste do template, e o custo é único (na escrita do ADR), não recorrente.
- **Risco de "gatilho artificial"** — alguém com vontade de implementar pode forjar evidência pra disparar o gatilho. Mitigação: gatilhos são verificáveis (data + log + binário), e a promoção exige ADR próprio que re-avalia a evidência.
- **Templates viram boilerplate** — se todos os ADRs tiverem `## Deferral` com mesma cara, a leitura pode ficar mecânica. Mitigação: conteúdo dos gatilhos e proibições é específico ao domínio; só a estrutura é reusada.

### Neutral

- **Não afeta ADRs sem Plano B** — a metodologia só se aplica quando o ADR de fato adia algo. ADRs de feature única (sem fallback opcional) ficam intactos.
- **ADR-038 já pratica parte do framework** — scope reduction + deferred tag. ADR-046 formaliza a estrutura transversal; ADR-038 não precisa ser renumerado.
- **Cross-link com ADR-045 (first application)** — qualquer leitura de ADR-046 deve apontar pra ADR-045 como exemplo concreto.

## Compatibility

- **ADRs existentes que usam "opcional"**: aplicar retroativamente quando fizer sentido (ADR-035 embedder fallback, ADR-042 workers sidecar). Não é breaking — a seção nova adiciona informação sem contradizer a anterior.
- **ADRs já Accepted**: revisão opcional; só aplicar `## Deferral` quando o autor do ADR estiver revisando por outro motivo.
- **ADRs Proposed novos**: `## Deferral` é mandatório quando o ADR adiar Plano B. Reviewer verifica presença antes de aceitar.
- **Specs `tlc-spec-driven`**: a fase "Specify" deve listar explicitamente "Planos B deferred" no escopo, linkando pro ADR de origem.

## Implementation Plan

1. ✅ Retrofit ADR-045 (commit `64711c7`) — primeira aplicação.
2. Aplicar retroativamente em ADR-035 (embedder fallback) e ADR-042 (workers sidecar) na próxima revisão de cada um.
3. Adicionar este ADR ao `docs/adr/README.md` índice.
4. Adicionar referência em `docs/REFERENCES.md` (entrada "Decisões Arquiteturais").
5. Incluir na revisão de novos ADRs como checklist do reviewer: "Se ADR propõe Plano B, exige seção `## Deferral` com 4 subseções?".

## References

- **ADR-038** — Viewer com Site Único e Dataset Fixo (B3 do ADR-036 revertido). Pratica "scope reduction + deferred" como decisão de produto; ADR-046 eleva o mesmo princípio pra metodologia transversal.
- **ADR-045** — ASR Streaming Engine (Nemotron 3.5 via ONNX Runtime Go). First application do framework `## Deferral: Plano B (Python Sidecar)`. Commit `64711c7` (2026-09-19).
- **ADR-035** — Embedder Embutido com Fallback ONNX MiniLM. Próximo candidato a retrofit do framework.
- **ADR-042** — mymemoryd com Workers Sidecar Isolados. Próximo candidato a retrofit (TTS/LLM podem continuar ativos; ASR sidecar pode virar `## Deferral`).
- **Pesquisa autoral `pesquisa-infraestrutura-autoral-mymemory.md`** §6 (escolhas tecnológicas) — base filosófica YAGNI + scope reduction.
- **`tlc-spec-driven`** skill — rito de promoção de Plano B quando gatilho dispara.
- **Princípio YAGNI geral** — "You Aren't Gonna Need It" (XP). Base filosófica do framework.
