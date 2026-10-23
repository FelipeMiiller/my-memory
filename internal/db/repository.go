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
