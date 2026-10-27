package pgtest

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"

	"github.com/lib/pq"
)

// Recorder é um driver database/sql que delega ao lib/pq e grava o texto SQL
// EXATO que chega ao servidor.
//
// POR QUE ELE EXISTE: a conversão "?" -> "$N" acontece dentro do processo
// (dialect.Bind -> bindOrdinal) e NADA no SQL enviado ao Postgres mostra que
// ela foi feita — o servidor só recebe "$1". Para provar a conversão em
// runtime real é preciso ver o texto que SAI. Este recorder é a costura: o SQL
// e os argumentos são exatamente os do caminho de produção; só a conexão é
// instrumentada.
//
// O recorder NÃO altera a query: ele a repassa intacta ao lib/pq. Qualquer
// diferença de comportamento entre o caminho instrumentado e o de produção
// seria um bug do recorder, não do código sob teste.
type Recorder struct {
	mu      sync.Mutex
	queries []string
}

// NewRecorder devolve um recorder vazio.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// OpenDB abre uma conexão instrumentada contra a DSN informada.
func (r *Recorder) OpenDB(dsn string) (*sql.DB, error) {
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(&recordingConnector{Connector: connector, rec: r}), nil
}

// Queries devolve uma cópia das queries gravadas, na ordem de chegada.
func (r *Recorder) Queries() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.queries))
	copy(out, r.queries)
	return out
}

// Len devolve quantas queries foram gravadas.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.queries)
}

// Reset descarta as queries gravadas.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries = nil
}

func (r *Recorder) record(query string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries = append(r.queries, query)
}

// recordingConnector embrulha o connector do lib/pq.
type recordingConnector struct {
	driver.Connector
	rec *Recorder
}

func (c *recordingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &recordingConn{Conn: conn, rec: c.rec}, nil
}

func (c *recordingConnector) Driver() driver.Driver {
	return c.Connector.Driver()
}

// recordingConn embrulha a conexão do lib/pq e grava o SQL que passa por ela.
type recordingConn struct {
	driver.Conn
	rec *Recorder
}

func (c *recordingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.rec.record(query)
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, errors.New("pgtest: a conexao do lib/pq nao implementa driver.ExecerContext")
	}
	return execer.ExecContext(ctx, query, args)
}

func (c *recordingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.rec.record(query)
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, errors.New("pgtest: a conexao do lib/pq nao implementa driver.QueryerContext")
	}
	return queryer.QueryContext(ctx, query, args)
}

// Prepare grava a query também: o database/sql cai no caminho Prepare quando
// a conexão não implementa ExecerContext, e o SQL que importa é o mesmo.
func (c *recordingConn) Prepare(query string) (driver.Stmt, error) {
	c.rec.record(query)
	return c.Conn.Prepare(query)
}

// CheckNamedValue delega ao lib/pq: o driver aceita qualquer tipo (inclusive
// sql.NullString) e o database/sql não deve converter nada por conta própria.
func (c *recordingConn) CheckNamedValue(nv *driver.NamedValue) error {
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(nv)
	}
	return nil
}
