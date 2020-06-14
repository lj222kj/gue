package pgxv4

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/vgarvardt/gue/adapter"
)

// Row implements adapter.Row using github.com/jackc/pgx/v4
type Row struct {
	row pgx.Row
}

// Scan implements adapter.Row.Scan() using github.com/jackc/pgx/v4
func (r *Row) Scan(dest ...interface{}) error {
	err := r.row.Scan(dest...)
	if err == pgx.ErrNoRows {
		return adapter.ErrNoRows
	}

	return err
}

// CommandTag implements adapter.CommandTag using github.com/jackc/pgx/v4
type CommandTag struct {
	ct pgconn.CommandTag
}

// RowsAffected implements adapter.CommandTag.RowsAffected() using github.com/jackc/pgx/v4
func (ct CommandTag) RowsAffected() int64 {
	return ct.ct.RowsAffected()
}

// Tx implements adapter.Tx using github.com/jackc/pgx/v4
type Tx struct {
	tx pgx.Tx
}

// Exec implements adapter.Tx.Exec() using github.com/jackc/pgx/v4
func (tx *Tx) Exec(ctx context.Context, sql string, arguments ...interface{}) (adapter.CommandTag, error) {
	ct, err := tx.tx.Exec(ctx, sql, arguments...)
	return CommandTag{ct}, err
}

// QueryRow implements adapter.Tx.QueryRow() using github.com/jackc/pgx/v4
func (tx *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) adapter.Row {
	return &Row{tx.tx.QueryRow(ctx, sql, args...)}
}

// Rollback implements adapter.Tx.Rollback() using github.com/jackc/pgx/v4
func (tx *Tx) Rollback(ctx context.Context) error {
	err := tx.tx.Rollback(ctx)
	if err == pgx.ErrTxClosed {
		return adapter.ErrTxClosed
	}

	return err
}

// Commit implements adapter.Tx.Commit() using github.com/jackc/pgx/v4
func (tx *Tx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

// Conn implements adapter.Conn using github.com/jackc/pgx/v4
type Conn struct {
	conn *pgx.Conn
}

// NewConn instantiates new adapter.Conn using github.com/jackc/pgx/v4
func NewConn(conn *pgx.Conn) adapter.Conn {
	return &Conn{conn}
}

// Exec implements adapter.Conn.Exec() using github.com/jackc/pgx/v4
func (c *Conn) Exec(ctx context.Context, sql string, arguments ...interface{}) (adapter.CommandTag, error) {
	ct, err := c.conn.Exec(ctx, sql, arguments...)
	return CommandTag{ct}, err
}

// QueryRow implements adapter.Conn.QueryRow() using github.com/jackc/pgx/v4
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...interface{}) adapter.Row {
	return &Row{c.conn.QueryRow(ctx, sql, args...)}
}

// Begin implements adapter.Conn.Begin() using github.com/jackc/pgx/v4
func (c *Conn) Begin(ctx context.Context) (adapter.Tx, error) {
	tx, err := c.conn.Begin(ctx)
	return &Tx{tx}, err
}

// Close implements adapter.Conn.Close() using github.com/jackc/pgx/v4
func (c *Conn) Close(ctx context.Context) error {
	return c.conn.Close(ctx)
}

// PoolConn implements adapter.Conn using github.com/jackc/pgx/v4,
// used for wrapping pool connection.
type PoolConn struct {
	conn *pgxpool.Conn
}

// Exec implements adapter.Conn.Exec() using github.com/jackc/pgx/v4
func (c *PoolConn) Exec(ctx context.Context, sql string, arguments ...interface{}) (adapter.CommandTag, error) {
	ct, err := c.conn.Exec(ctx, sql, arguments...)
	return CommandTag{ct}, err
}

// QueryRow implements adapter.Conn.QueryRow() using github.com/jackc/pgx/v4
func (c *PoolConn) QueryRow(ctx context.Context, sql string, args ...interface{}) adapter.Row {
	return &Row{c.conn.QueryRow(ctx, sql, args...)}
}

// Begin implements adapter.Conn.Begin() using github.com/jackc/pgx/v4
func (c *PoolConn) Begin(ctx context.Context) (adapter.Tx, error) {
	tx, err := c.conn.Begin(ctx)
	return &Tx{tx}, err
}

// Close implements adapter.Conn.Close() using github.com/jackc/pgx/v4
func (c *PoolConn) Close(ctx context.Context) error {
	c.conn.Release()
	return nil
}

// ConnPool implements adapter.ConnPool using github.com/jackc/pgx/v4
type ConnPool struct {
	pool *pgxpool.Pool
}

// NewConnPool instantiates new adapter.ConnPool using github.com/jackc/pgx/v4
func NewConnPool(pool *pgxpool.Pool) adapter.ConnPool {
	return &ConnPool{pool}
}

// Exec implements adapter.ConnPool.Exec() using github.com/jackc/pgx/v4
func (c *ConnPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (adapter.CommandTag, error) {
	ct, err := c.pool.Exec(ctx, sql, arguments...)
	return CommandTag{ct}, err
}

// QueryRow implements adapter.ConnPool.QueryRow() using github.com/jackc/pgx/v4
func (c *ConnPool) QueryRow(ctx context.Context, sql string, args ...interface{}) adapter.Row {
	return &Row{c.pool.QueryRow(ctx, sql, args...)}
}

// Begin implements adapter.ConnPool.Begin() using github.com/jackc/pgx/v4
func (c *ConnPool) Begin(ctx context.Context) (adapter.Tx, error) {
	tx, err := c.pool.Begin(ctx)
	return &Tx{tx}, err
}

// Acquire implements adapter.ConnPool.Acquire() using github.com/jackc/pgx/v4
func (c *ConnPool) Acquire(ctx context.Context) (adapter.Conn, error) {
	conn, err := c.pool.Acquire(ctx)
	return &PoolConn{conn}, err
}

// Release implements adapter.ConnPool.Release() using github.com/jackc/pgx/v4
func (c *ConnPool) Release(conn adapter.Conn) {
	conn.(*PoolConn).conn.Release()
}

// Stat implements adapter.ConnPool.Stat() using github.com/jackc/pgx/v4
func (c *ConnPool) Stat() adapter.ConnPoolStat {
	s := c.pool.Stat()
	return adapter.ConnPoolStat{
		MaxConnections:       int(s.MaxConns()),
		CurrentConnections:   int(s.TotalConns()),
		AvailableConnections: int(s.TotalConns() - s.IdleConns()),
	}
}

// Close implements adapter.ConnPool.Close() using github.com/jackc/pgx/v4
func (c *ConnPool) Close() {
	c.pool.Close()
}
