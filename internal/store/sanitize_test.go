package store

import "testing"

func TestSanitizePostgresURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "URL completa com user, pass, host, port, path, query",
			in:   "postgres://app_user:s3cr3t@db.internal:5432/my_memory?sslmode=require",
			want: "postgres://***@db.internal:5432/my_memory?sslmode=require",
		},
		{
			name: "URL sem credenciais permanece inalterada",
			in:   "postgres://localhost/db",
			want: "postgres://localhost/db",
		},
		{
			name: "URL só com user (sem senha) — mascara user também",
			in:   "postgres://app_user@db.host:5432/db",
			want: "postgres://***@db.host:5432/db",
		},
		{
			name: "URL vazia retorna vazia",
			in:   "",
			want: "",
		},
		{
			name: "URL com senha containing special chars",
			in:   "postgres://user:p%40ssw0rd@db.host:5432/db",
			want: "postgres://***@db.host:5432/db",
		},
		{
			name: "URL inválida usa fallback",
			in:   "not-a-valid-url://anything:bad@host:123/db",
			want: "not-a-valid-url://***@host:123/db",
		},
		{
			name: "URL sem @ permanece (sem credenciais pra mascarar)",
			in:   "postgres://localhost:5432/db",
			want: "postgres://localhost:5432/db",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizePostgresURL(tc.in)
			if got != tc.want {
				t.Errorf("sanitizePostgresURL(%q):\n  got  = %s\n  want = %s", tc.in, got, tc.want)
			}
		})
	}
}
