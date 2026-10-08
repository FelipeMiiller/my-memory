package store

import (
	"strings"
	"testing"
)

func TestComposeConnString(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		password string
		want     string
	}{
		{
			name:     "URL vazia devolve vazia",
			url:      "",
			password: "secret",
			want:     "",
		},
		{
			name:     "senha vazia nao altera a URL",
			url:      "postgres://u@h:5432/db?sslmode=require",
			password: "",
			want:     "postgres://u@h:5432/db?sslmode=require",
		},
		{
			name:     "usuario sem senha recebe a senha",
			url:      "postgres://default@h:5432/db",
			password: "secret",
			want:     "postgres://default:secret@h:5432/db",
		},
		{
			name:     "URL que ja tem senha nao e sobrescrita",
			url:      "postgres://default:old@h:5432/db",
			password: "new",
			want:     "postgres://default:old@h:5432/db",
		},
		{
			name:     "sem userinfo vira query param password",
			url:      "postgres://h:5432/db",
			password: "secret",
			want:     "postgres://h:5432/db?password=secret",
		},
		{
			name:     "query params existentes sao preservados",
			url:      "postgres://u@h:5432/db?sslmode=require&channel_binding=require",
			password: "secret",
			want:     "postgres://u:secret@h:5432/db?sslmode=require&channel_binding=require",
		},
		{
			name:     "DSN key=value nao parseavel volta intacto",
			url:      "host=h port=5432 dbname=db",
			password: "secret",
			want:     "host=h port=5432 dbname=db",
		},
		{
			name:     "senha com caracteres especiais e escapada",
			url:      "postgres://default@h:5432/db",
			password: "p@ss w0rd",
			want:     "postgres://default:p%40ss%20w0rd@h:5432/db",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComposeConnString(tc.url, tc.password)
			if got != tc.want {
				t.Errorf("ComposeConnString(%q, %q)\n  got  = %q\n  want = %q", tc.url, tc.password, got, tc.want)
			}
		})
	}
}

// O segredo nunca pode vazar em log/diagnóstico.
func TestComposeConnString_NaoVazaSenhaEmLog(t *testing.T) {
	url := "postgres://default:h:5432/db"
	composed := ComposeConnString(url, "TOPSECRET123")
	if !strings.Contains(composed, "TOPSECRET123") {
		t.Fatal("a senha deveria estar na string composta")
	}
	masked := sanitizePostgresURL(composed)
	if strings.Contains(masked, "TOPSECRET123") {
		t.Errorf("sanitizePostgresURL vazou a senha: %s", masked)
	}
}
