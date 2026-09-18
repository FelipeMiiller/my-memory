package main

import "testing"

// TestApplyStorageOverride cobre ADR-040 EARS-4 e EARS-5:
// - override vazio → passa-through (no-op)
// - override=sqlite → força SQLite, ignora Postgres
// - override=postgres → exige URL, erro explícito se ausente
// - override inválido → erro de validação
func TestApplyStorageOverride(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		resolvedDB  string
		resolvedPG  string
		wantDB      string
		wantPG      string
		wantErr     bool
		description string
	}{
		{
			name:        "override vazio = no-op",
			override:    "",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "",
			wantDB:      ".memory/memory.db",
			wantPG:      "",
			description: "Sem flag, mantém decisão do resolver",
		},
		{
			name:        "override=sqlite sem Postgres = no-op",
			override:    "sqlite",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "",
			wantDB:      ".memory/memory.db",
			wantPG:      "",
			description: "Forçar sqlite quando já era sqlite: idempotente",
		},
		{
			name:        "override=sqlite com Postgres = zera pg",
			override:    "sqlite",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "postgres://localhost/db",
			wantDB:      ".memory/memory.db",
			wantPG:      "",
			description: "EARS-4: --storage=sqlite ignora Postgres detectado",
		},
		{
			name:        "override=postgres com URL = usa URL",
			override:    "postgres",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "postgres://localhost/db",
			wantDB:      "",
			wantPG:      "postgres://localhost/db",
			description: "Override consciente: db zera, pg é a URL",
		},
		{
			name:        "override=postgres sem URL = erro",
			override:    "postgres",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "",
			wantErr:     true,
			description: "EARS-5: erro explícito quando --storage=postgres sem URL",
		},
		{
			name:        "override inválido = erro",
			override:    "mysql",
			resolvedDB:  ".memory/memory.db",
			resolvedPG:  "",
			wantErr:     true,
			description: "Validação de input: só aceita 'sqlite' ou 'postgres'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDB, gotPG, err := applyStorageOverride(tc.override, tc.resolvedDB, tc.resolvedPG)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("%s\n  esperava erro, obteve nil", tc.description)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s\n  erro inesperado: %v", tc.description, err)
			}
			if gotDB != tc.wantDB {
				t.Errorf("%s\n  db  got=%q want=%q", tc.description, gotDB, tc.wantDB)
			}
			if gotPG != tc.wantPG {
				t.Errorf("%s\n  pg  got=%q want=%q", tc.description, gotPG, tc.wantPG)
			}
		})
	}
}
