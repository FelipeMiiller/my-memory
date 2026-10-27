// Package pgtest é o suporte compartilhado dos testes de integração PostgreSQL.
//
// Ele existe porque três binários de teste (internal/db, internal/store e
// cmd/mem) precisam das mesmas duas coisas — e antes delas cada um fazia do
// seu jeito, o do internal/store do jeito ERRADO:
//
//  1. A TRAVA que impede o teste de escrever no Postgres real do dono.
//     MY_MEMORY_PG_URL (e as outras variáveis de storage) fazem o store
//     rotear para o Postgres configurado. Sem neutralizá-las, um `go test ./...`
//     com a variável no ambiente grava lixo no banco de produção — foi o que a
//     ISSUE-031 mediu (80 repositórios e 362 documentos no banco do dono).
//     O padrão já existe em cmd/mem/main_test.go e internal/config/main_test.go:
//     capturar o valor, neutralizar, restaurar no fim.
//
//  2. O BANCO DESCARTÁVEL contra o qual a integração roda. O padrão já existe
//     em internal/db/code_bridge_postgres_test.go: CREATE DATABASE
//     mymemory_pgtest_<pid>_<nanos>, aplicar o schema, asserir, DROP DATABASE
//     ... WITH (FORCE). O banco `mymemory` do dono NUNCA é o alvo.
//
// Concentrar os dois aqui evita que um teste novo reintroduza o vazamento por
// descuido: o único caminho para o Postgres em um teste de integração é
// pgtest.New, que cria o banco descartável e o derruba no fim.
package pgtest

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// OptInEnvVar é a variável que habilita a integração PostgreSQL.
//
// Sem ela a suíte padrão é hermética: nenhum teste de integração roda, nenhum
// banco é criado e nada toca o Postgres do dono.
const OptInEnvVar = "MY_MEMORY_PG_TEST"

// StorageEnvVars são as variáveis que fazem o store/CLI rotearem para um
// Postgres externo. São neutralizadas durante a suíte.
var StorageEnvVars = []string{
	"MY_MEMORY_PG_URL",
	"MY_MEMORY_PG_PASSWORD",
	"POSTGRES_URL",
	"DATABASE_URL",
}

var (
	mu       sync.Mutex
	adminURL string
)

// Capture guarda a URL do Postgres que está no ambiente ANTES da neutralização.
//
// A captura precisa acontecer antes: depois de Neutralize a variável some do
// ambiente e o teste de integração não tem como chegar ao servidor para criar
// o banco descartável.
//
// Se Capture não for chamada, AdminURL devolve "" e todo teste de integração
// é PULADO — a falha cai para o lado seguro (nada é escrito), nunca para o
// lado perigoso.
func Capture() {
	mu.Lock()
	defer mu.Unlock()
	adminURL = strings.TrimSpace(os.Getenv("MY_MEMORY_PG_URL"))
}

// OptedIn reporta se a integração PostgreSQL está habilitada.
func OptedIn() bool {
	return strings.TrimSpace(os.Getenv(OptInEnvVar)) == "1"
}

// AdminURL devolve a URL capturada do Postgres, ou "" se a integração não está
// habilitada (ou se Capture não foi chamada).
func AdminURL() string {
	if !OptedIn() {
		return ""
	}
	mu.Lock()
	defer mu.Unlock()
	return adminURL
}

// Neutralize remove do ambiente as variáveis de storage e devolve a função que
// restaura os valores originais.
//
// O padrão é o de cmd/mem/main_test.go: as variáveis são capturadas antes e
// restauradas no fim, para não afetar a sessão de quem chamou `go test`.
func Neutralize() func() {
	saved := map[string]string{}
	for _, k := range StorageEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
		}
		if err := os.Unsetenv(k); err != nil {
			panic("pgtest: nao foi possivel limpar " + k + ": " + err.Error())
		}
	}
	return func() {
		for k, v := range saved {
			_ = os.Setenv(k, v)
		}
	}
}

// Harness é o banco PostgreSQL descartável de um teste.
type Harness struct {
	// AdminURL é a URL do servidor (usada só para criar e derrubar o banco).
	AdminURL string
	// DBName é o nome do banco descartável criado para o teste.
	DBName string
	// DSN é a URL de conexão do banco descartável.
	DSN string
}

// New cria um banco PostgreSQL descartável para o teste t e o derruba quando o
// teste termina.
//
// Ele PULA o teste quando a integração não está habilitada: sem
// MY_MEMORY_PG_TEST=1 nenhum banco é criado e nada toca o Postgres do dono.
func New(t testing.TB) *Harness {
	t.Helper()

	admin := AdminURL()
	if admin == "" {
		t.Skipf("integracao PostgreSQL desligada: defina %s=1 (e MY_MEMORY_PG_URL) para habilitar", OptInEnvVar)
		return nil
	}

	name := fmt.Sprintf("mymemory_pgtest_%d_%d", os.Getpid(), time.Now().UnixNano()%1000000)
	dsn, err := createDatabase(admin, name)
	if err != nil {
		t.Fatalf("pgtest: CREATE DATABASE %s falhou: %v", name, err)
		return nil
	}

	h := &Harness{AdminURL: admin, DBName: name, DSN: dsn}
	t.Cleanup(func() {
		if err := dropDatabase(admin, name); err != nil {
			t.Errorf("pgtest: DROP DATABASE %s falhou: %v", name, err)
		}
	})
	return h
}

// createDatabase cria o banco descartável e devolve a DSN dele.
func createDatabase(adminURL, name string) (string, error) {
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return "", err
	}
	defer admin.Close()

	if err := admin.Ping(); err != nil {
		return "", err
	}
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		return "", err
	}
	return databaseURL(adminURL, name)
}

// dropDatabase derruba o banco descartável. WITH (FORCE) desconecta as
// sessões abertas (o teste pode ter deixado alguma conexão viva).
func dropDatabase(adminURL, name string) error {
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return err
	}
	defer admin.Close()

	_, err = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
	return err
}

// databaseURL aponta a mesma URL (mesmo host, credenciais e sslmode) para
// outro banco.
func databaseURL(adminURL, name string) (string, error) {
	u, err := url.Parse(adminURL)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}
