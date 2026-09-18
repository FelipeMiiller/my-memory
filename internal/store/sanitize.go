package store

import (
	"net/url"
	"strings"
)

// sanitizePostgresURL mascara credenciais em URLs PostgreSQL antes de logar.
// Substitui `user:password@host` por `***@host` para evitar vazamento de credenciais
// em logs, mensagens de erro e stack traces.
//
// Exemplos:
//
//	postgres://user:s3cr3t@db.host:5432/mydb?sslmode=require
//	  → postgres://***@db.host:5432/mydb?sslmode=require
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
		return fallbackSanitizeURL(rawURL)
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
	return result
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
