package store

import (
	"net/url"
	"strings"
)

// sanitizePostgresURL mascara credenciais em URLs PostgreSQL antes de logar.
// Substitui `user:password@host` por `***@host` para evitar vazamento de credenciais
// em logs, mensagens de erro e stack traces. Também mascara `?password=` na query,
// que é o formato usado quando a credencial vem separada da URL.
//
// Exemplos:
//
//	postgres://user:s3cr3t@db.host:5432/mydb?sslmode=require
//	  → postgres://***@db.host:5432/mydb?sslmode=require
//
//	postgres://db.host:5432/mydb?password=s3cr3t&sslmode=require
//	  → postgres://db.host:5432/mydb?password=***&sslmode=require
//
//	postgres://localhost/db  (sem credenciais)
//	  → postgres://localhost/db  (inalterada)
//
// Se a URL for inválida, retorna versão mascarada baseada em busca de padrão `user:pass@host`
// sem usar net/url (fallback seguro).
func sanitizePostgresURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		// Fallback: regex simples pra `scheme://user:pass@host`
		return maskPasswordQueryParam(fallbackSanitizeURL(rawURL))
	}

	host := parsed.Host
	if atIdx := strings.Index(host, "@"); atIdx >= 0 {
		host = host[atIdx+1:]
	}

	masked := *parsed
	masked.User = nil
	masked.Host = host

	// Mantém scheme + path + query, apenas mascara credenciais.
	result := masked.String()
	if parsed.User != nil {
		// Garante formato `scheme://***@host:port/path?query`
		if parsed.Scheme != "" {
			result = parsed.Scheme + "://***@" + host + parsed.Path
			if parsed.RawQuery != "" {
				result += "?" + parsed.RawQuery
			}
		}
	}
	return maskPasswordQueryParam(result)
}

// maskPasswordQueryParam substitui o valor de `password=` na query por `***`.
// Necessário porque ComposeConnString injeta a senha como query param quando a
// URL não tem userinfo — sem isso a senha apareceria em erro de conexão.
func maskPasswordQueryParam(s string) string {
	const key = "password="
	idx := strings.Index(s, key)
	if idx < 0 {
		return s
	}
	start := idx + len(key)
	end := strings.IndexAny(s[start:], "&")
	if end < 0 {
		return s[:start] + "***"
	}
	return s[:start] + "***" + s[start+end:]
}

// sanitizeConnString mascara credenciais tanto em URL quanto em DSN key=value.
// BuildDSN produz DSNs, então é esta função — não sanitizePostgresURL — que
// protege os erros de conexão no caminho em uso.
func sanitizeConnString(s string) string {
	if strings.Contains(s, "://") {
		return sanitizePostgresURL(s)
	}
	return maskDSNPassword(s)
}

// maskDSNPassword substitui o valor do campo `password=` por `***`, lida com
// aspas simples e barra invertida do formato lib/pq.
func maskDSNPassword(dsn string) string {
	lower := strings.ToLower(dsn)
	idx := strings.Index(lower, "password=")
	if idx < 0 {
		return dsn
	}
	start := idx + len("password=")
	end := strings.IndexAny(dsn[start:], " \t")
	valEnd := start + end
	if end < 0 {
		valEnd = len(dsn)
	}
	return dsn[:start] + "***" + dsn[valEnd:]
}

// fallbackSanitizeURL faz mascaramento via regex quando net/url falha.
// Procura padrão `scheme://anything@host` e substitui `anything` por `***`.
func fallbackSanitizeURL(rawURL string) string {
	// Padrão: tudo entre `://` e `@` é considerado credencial.
	start := strings.Index(rawURL, "://")
	if start < 0 {
		return rawURL
	}
	rest := rawURL[start+3:]
	atIdx := strings.Index(rest, "@")
	if atIdx < 0 {
		return rawURL
	}
	return rawURL[:start+3] + "***@" + rest[atIdx+1:]
}
