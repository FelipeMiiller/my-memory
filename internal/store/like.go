package store

import "strings"

// LikeEscapeChar é o caractere de escape usado nas cláusulas LIKE dos dois
// engines. Precisa ser explícito: o SQLite não tem caractere de escape padrão
// para o LIKE (a barra invertida é literal), enquanto o Postgres usa a barra por
// padrão. Declarar o mesmo nos dois é o que mantém `mem read` se comportando
// igual em qualquer vault.
const LikeEscapeChar = "\\"

// EscapeLikePattern escapa os curingas do LIKE para que o argumento do usuário
// seja comparado literalmente.
//
// Sem isso, um nome com `_` casa com qualquer caractere e um nome com `%` casa
// com qualquer sequência: `mem read my_note.md` encontraria `myXnote.md` e
// acusaria ambiguidade onde não existe nenhuma. O nome do arquivo é dado, não
// padrão.
//
// strings.NewReplacer reescreve em uma passada só, então a barra invertida que
// inserimos aqui não é reescapada pelas próprias substituições seguintes.
func EscapeLikePattern(s string) string {
	return strings.NewReplacer(
		LikeEscapeChar, LikeEscapeChar+LikeEscapeChar,
		"%", LikeEscapeChar+"%",
		"_", LikeEscapeChar+"_",
	).Replace(s)
}
