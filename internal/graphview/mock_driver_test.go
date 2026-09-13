package graphview

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
)

var (
	registerOnce sync.Once
	mockQueryMu  sync.Mutex
	mockQueryFn  func(query string) (driver.Rows, error)
)

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare not implemented")
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return nil, errors.New("begin not implemented")
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	mockQueryMu.Lock()
	defer mockQueryMu.Unlock()

	if mockQueryFn != nil {
		return mockQueryFn(query)
	}
	return nil, errors.New("mockQueryFn not defined")
}

var _ driver.QueryerContext = (*mockConn)(nil)

type mockRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *mockRows) Columns() []string {
	return r.columns
}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i, val := range r.rows[r.idx] {
		dest[i] = val
	}
	r.idx++
	return nil
}

func setupMockDB() (*sql.DB, error) {
	registerOnce.Do(func() {
		sql.Register("mock_sqlite", &mockDriver{})
	})
	return sql.Open("mock_sqlite", "test")
}
