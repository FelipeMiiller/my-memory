package db

import (
	"context"
	"strings"
)

// DefaultRepository é o repositório usado quando nenhum repo_id foi
// resolvido para o vault (o mesmo default do store de notas).
const DefaultRepository = "default"

// repositoryCtxKey é a chave (não exportada) do repo_id no contexto.
type repositoryCtxKey struct{}

// WithRepository devolve um contexto que carrega o repo_id resolvido do
// vault.
//
// O code pipeline (internal/codeast) grava code_files/code_symbols/
// code_edges através dos helpers de internal/db — e o `repository` gravado
// precisa ser o do vault, não um global. O contexto é o canal que atravessa
// o pipeline sem depender de assinatura nova nos helpers.
func WithRepository(ctx context.Context, repository string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	repository = strings.TrimSpace(repository)
	if repository == "" {
		repository = DefaultRepository
	}
	return context.WithValue(ctx, repositoryCtxKey{}, repository)
}

// RepositoryFromContext devolve o repo_id carregado pelo contexto.
// Sem repo_id no contexto, devolve DefaultRepository.
func RepositoryFromContext(ctx context.Context) string {
	if ctx == nil {
		return DefaultRepository
	}
	if repo, ok := ctx.Value(repositoryCtxKey{}).(string); ok && strings.TrimSpace(repo) != "" {
		return repo
	}
	return DefaultRepository
}

// astHashKey é a chave (não exportada) da política de ast_hash no contexto.
type astHashKey struct{}

// WithAstHash carrega no contexto a política de armazenamento do ast_hash.
//
// Com enabled = false, o code_files guarda ast_hash NULL: o cache
// incremental perde a política "conteúdo mudou, AST não mudou" e reparseia
// o arquivo. É o que a flag `--ast-hash=false` promete.
func WithAstHash(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, astHashKey{}, enabled)
}

// AstHashEnabled devolve false quando o contexto pede para não armazenar o
// ast_hash. O default é true (o comportamento histórico).
func AstHashEnabled(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	if enabled, ok := ctx.Value(astHashKey{}).(bool); ok {
		return enabled
	}
	return true
}
