package db

import (
	"database/sql"
)

// sql 错误时直接 panic, 由框架错误处理逻辑回收
// sql.ErrNoRows 认为是正确返回 0 行，不抛错
var PANIC_DB_ERROR = false

type Executor interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type DbExecutor struct {
	*sql.DB
}

type TxExecutor struct {
	*sql.Tx
}

func errorHandle(err error) error { //{{{
	if PANIC_DB_ERROR {
		panic(err)
	}

	return err
} // }}}
