package store

import (
	"strings"
	"testing"
)

func TestBuildDSN(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		password string
		want     []string // campos que precisam estar presentes
		absent   []string // campos que NÃO podem aparecer
	}{
		{
			name:     "URL sem credencial vira DSN com password separado",
			url:      "postgres://default@h:5432/db?sslmode=require",
			password: "secret",
			want:     []string{"host=h", "port=5432", "dbname=db", "user=default", "password=secret", "sslmode=require"},
			absent:   []string{"://"},
		},
		{
			name:     "senha embutida na URL e extraida para campo",
			url:      "postgres://default:old@h:5432/db",
			password: "",
			want:     []string{"host=h", "user=default", "password=old"},
			absent:   []string{"://", "default:old@"},
		},
		{
			name:     "campo password tem precedencia sobre a URL",
			url:      "postgres://default:old@h:5432/db",
			password: "new",
			want:     []string{"password=new"},
			absent:   []string{"password=old"},
		},
		{
			name:     "varios query params preservados",
			url:      "postgres://u@h/db?sslmode=require&channel_binding=require",
			password: "secret",
			want:     []string{"sslmode=require", "channel_binding=require", "password=secret"},
		},
		{
			name:     "DSN existente recebe password",
			url:      "host=h port=5432 dbname=db user=u",
			password: "secret",
			want:     []string{"host=h", "password=secret"},
		},
		{
			name:     "DSN existente com password nao e alterado",
			url:      "host=h dbname=db password=old",
			password: "new",
			want:     []string{"password=old"},
			absent:   []string{"password=new"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dsn := BuildDSN(tc.url, tc.password)
			for _, w := range tc.want {
				if !strings.Contains(dsn, w) {
					t.Errorf("DSN %q não contém %q", dsn, w)
				}
			}
			for _, a := range tc.absent {
				if strings.Contains(dsn, a) {
					t.Errorf("DSN %q não deveria conter %q", dsn, a)
				}
			}
		})
	}
}

// A separação só é real se a senha nunca virar URL.
func TestBuildDSN_SenhaNuncaViraURL(t *testing.T) {
	url := "postgres://default@h:5432/db?sslmode=require"
	dsn := BuildDSN(url, "TOPSECRET123")

	if strings.Contains(dsn, "://") {
		t.Fatalf("DSN ainda parece URL: %q", dsn)
	}
	if !strings.Contains(dsn, "password=TOPSECRET123") {
		t.Fatalf("a senha deveria estar como campo password: %q", dsn)
	}
	if masked := sanitizeConnString(dsn); strings.Contains(masked, "TOPSECRET123") {
		t.Errorf("sanitizeConnString vazou a senha: %q", masked)
	}
}

// Senha com caractere especial precisa sobreviver ao DSN.
func TestBuildDSN_SenhaComCaractereEspecial(t *testing.T) {
	dsn := BuildDSN("postgres://default@h/db", "p@ss w0rd")
	if !strings.Contains(dsn, `password='p@ss w0rd'`) {
		t.Errorf("esperava valor com aspas simples, veio: %q", dsn)
	}
	if masked := sanitizeConnString(dsn); strings.Contains(masked, "p@ss") {
		t.Errorf("sanitize não mascarou valor entre aspas: %q", masked)
	}
}

func TestSanitizeConnString_URLEDSN(t *testing.T) {
	if got := sanitizeConnString("postgres://u:SECRET@h/db"); strings.Contains(got, "SECRET") {
		t.Errorf("URL não mascarada: %q", got)
	}
	if got := sanitizeConnString("host=h dbname=db password=SECRET user=u"); strings.Contains(got, "SECRET") {
		t.Errorf("DSN não mascarado: %q", got)
	}
}
