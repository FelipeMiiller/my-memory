# ADR-057: mtime e metadados de arquivo no índice

- **Data**: 2026-10-18
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: schema, metadados, staleness, obsidian

## Contexto e Problema

`documents.updated_at` guarda `time.Now().Unix()` no momento da indexação (`cmd/mem/main.go:1792`, `internal/watcher/indexer.go:93,228`). Ou seja: **é horário de indexação, não o mtime do arquivo.**

Isso é invisível enquanto ninguém pergunta "o que mudou recentemente", mas trava um conjunto de comandos:

| Comando do Obsidian | Por que está bloqueado |
|---|---|
| `file` (info: tamanho, criado, modificado) | `documents` não tem `size` nem `mtime`; `updated_at` mente |
| `recents` | Sem mtime persistido, só resta varrer o filesystem |
| `search` ordenado por relevância temporal | `--decay` (ADR-015) usa decay exponencial sobre `updated_at`, que reflete quando **indexamos**, não quando **editamos** |

O terceiro é o mais silencioso. `mem search --decay` promete priorizar notas recentes e hoje otimiza para "processadas por último". Uma nota editada há um mês e reindexada agora herda a mesma pontuação de decaimento que uma nota editada agora e nunca reindexada — desde que ambas entrem no mesmo `mem index`.

## Drivers da Decisão

- Separar **quando o documento mudou** de **quando o índice o viu**
- Habilitar `file`, `recents` e qualquer ordenação temporal correta
- Preservar `updated_at` com o significado atual (indexado em) ou redefini-lo sem quebrar consumers
- Detecção de staleness (ADR-028) precisa de mtime confiável

## Opções Consideradas

- **A — Coluna nova `mtime` ao lado de `updated_at` (escolhida)**
- **B — Redefinir `updated_at` para ser o mtime**
- **C — Não persistir; sempre varrer o disco**

## Resultado da Decisão

Escolhida a **Opção A**. `documents` ganha `mtime INTEGER` (epoch Unix, em segundos) e `size INTEGER`, ambos preenchidos no momento da leitura do arquivo. **`updated_at` mantém o significado de "quando o índice gravou"**, porque esse é o que o watchdog e o staleness já consultam.

A distinção fica explícita no schema:

- `mtime` — quando o autor editou o arquivo
- `updated_at` — quando o My-Memory processou o documento

Detecção de staleness passa a comparar `mtime` do disco contra `mtime` gravado, em vez de inferir mudança por `updated_at`.

### Consequências Positivas

- `file`, `recents` e ordenação temporal viram consultas triviais
- `--decay` passa a decayar sobre a edição real, que era a intenção da ADR-015
- Rename e move (ADR-054) podem detectar mudança de identidade por `mtime`
- Copy de vault preserva ordenação temporal entre máquinas

### Consequências Negativas

- Duas colunas de tempo com significados próximos convidem a bug — quem ler sem o comentário vai usar a errada
- `mem index` precisa do `os.Stat` por arquivo; hoje lê o conteúdo e faz hash, então o custo é marginal
- Vaults já indexados têm `mtime` nulo até a próxima indexação completa

## Por Que Não as Outras

### Opção B — Redefinir `updated_at` ❌

- ✅ Nenhuma coluna nova
- ❌ Quebra o consumo atual de `updated_at` no watcher e no staleness, queAssume que reflete gravação no índice
- ❌ Um `mem index --force` reescreveria todo `updated_at` do vault, destruindo a informação de "quando isso entrou no índice"

### Opção C — Varrer o disco ❌

- ✅ Zero migration
- ❌ `recents` em vault federado exigiria acesso ao filesystem de outro vault, que pode estar no Postgres e não em disco

## Nota de Implementação

A migração precisa preencher `mtime` a partir do arquivo em disco. Em vault já indexado o valor fica nulo até reindexação; `mem doctor` deve reportar a fração de documentos com `mtime` nulo, senão a divergência fica invisível.

## Links

- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — habilita `file` e `recents`
- [ADR-015](015-decaimento-temporal-exponencial-na-busca-hibrida.md) — o consumidor que hoje está errado
- [ADR-028](028-staleness-banners-e-deteccao-de-desatualizacao.md) — consumidor que passa a consumir `mtime`
- Evidência: `cmd/mem/main.go:1792`, `internal/db/schema.go:5-14`, `internal/staleness/detector.go:206,234`