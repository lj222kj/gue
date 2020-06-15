package testing

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib" // register pgx sql driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vgarvardt/gue/adapter/pgxv3"

	"github.com/vgarvardt/gue/adapter"
)

const defaultPoolConns = 5

var (
	applyMigrations sync.Once
)

// OpenTestPoolMaxConnsPGXv3 opens connections pool user in testing
func OpenTestPoolMaxConnsPGXv3(t testing.TB, maxConnections int) adapter.ConnPool {
	t.Helper()

	applyMigrations.Do(func() {
		doApplyMigrations(t)
	})

	connPoolConfig := pgx.ConnPoolConfig{
		ConnConfig:     testConnPGXv3Config(t),
		MaxConnections: maxConnections,
		AfterConnect:   pgxv3.PrepareStatements,
	}
	poolPGXv3, err := pgx.NewConnPool(connPoolConfig)
	require.NoError(t, err)

	pool := pgxv3.NewConnPool(poolPGXv3)

	t.Cleanup(func() {
		truncateAndClose(t, pool)
	})

	return pool
}

// OpenTestPoolPGXv3 opens connections pool user in testing
func OpenTestPoolPGXv3(t testing.TB) adapter.ConnPool {
	t.Helper()

	return OpenTestPoolMaxConnsPGXv3(t, defaultPoolConns)
}

// OpenTestConnPGXv3 opens connection user in testing
func OpenTestConnPGXv3(t testing.TB) adapter.Conn {
	t.Helper()

	conn, err := pgx.Connect(testConnPGXv3Config(t))
	require.NoError(t, err)

	return pgxv3.NewConn(conn)
}

func testConnDSN(t testing.TB) string {
	t.Helper()

	testPgConnString, found := os.LookupEnv("TEST_POSTGRES")
	require.True(t, found, "TEST_POSTGRES env var is not set")
	require.NotEmpty(t, testPgConnString, "TEST_POSTGRES env var is empty")

	//return `postgres://test:test@localhost:32769/test?sslmode=disable`
	return testPgConnString
}

func testConnPGXv3Config(t testing.TB) pgx.ConnConfig {
	t.Helper()

	cfg, err := pgx.ParseConnectionString(testConnDSN(t))
	require.NoError(t, err)

	return cfg
}

func truncateAndClose(t testing.TB, pool adapter.ConnPool) {
	t.Helper()

	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE que_jobs")
	assert.NoError(t, err)

	pool.Close()
}

func doApplyMigrations(t testing.TB) {
	t.Helper()

	migrationsConn, err := sql.Open("pgx", testConnDSN(t))
	require.NoError(t, err)
	defer func() {
		err := migrationsConn.Close()
		assert.NoError(t, err)
	}()

	migrationSQL, err := ioutil.ReadFile("./schema.sql")
	require.NoError(t, err)

	_, err = migrationsConn.Exec(string(migrationSQL))
	require.NoError(t, err)
}
