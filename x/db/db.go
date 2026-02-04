package db

import (
	"context"
	"database/sql"
	"strings"
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

type DBClient interface {
	SetDB(dbt string, dbo *sql.DB) error
	Ping(ctx context.Context) error
	Close()
	Type() string
	ID() string
	SetDebug(open bool)
	Begin(is_readonly bool) (DBClient, error)
	Rollback() error
	Commit() error
	Insert(table string, vals ...map[string]any) (int, error)
	Upsert(table string, vals map[string]any, ignore_fields ...string) (int, error)
	Update(table string, vals map[string]any, where string, val ...interface{}) (int, error)
	Delete(sqlOptions ...FnSqlOption) (int, error)
	Execute(query string, val ...any) (int, error)
	GetOne(sqlOptions ...FnSqlOption) (any, error)
	GetRow(sqlOptions ...FnSqlOption) (map[string]any, error)
	GetAll(sqlOptions ...FnSqlOption) ([]map[string]any, error)
	QueryOne(sqlstr string, vals ...any) (any, error)
	QueryRow(sqlstr string, vals ...any) (map[string]any, error)
	Query(sqlstr string, vals ...any) ([]map[string]any, error)
	QueryStream(sqlstr string, vals ...any) (*RowIter, error)
}

type FnSqlOption func(*SqlOption)

type SqlOption struct {
	table     string
	fields    string
	alias     string
	leftJoin  []string
	innerJoin []string
	idx       string
	group     string
	order     string
	limits    string
	where     string
	vals      []any
}

func (so *SqlOption) ToSql() (string, []any) { //{{{
	var sb strings.Builder

	if so.fields == "" {
		so.fields = "*"
	}

	sb.WriteString("SELECT ")
	sb.WriteString(so.fields)

	if so.table != "" {
		sb.WriteString(" FROM ")
		sb.WriteString(so.table)

		if so.alias != "" {
			sb.WriteString(" ")
			sb.WriteString(so.alias)
		}
	}

	if so.idx != "" {
		sb.WriteString(" FORCE INDEX (")
		sb.WriteString(so.idx)
		sb.WriteString(")")
	}

	if len(so.leftJoin) > 0 {
		for _, j := range so.leftJoin {
			sb.WriteString(" LEFT JOIN ")
			sb.WriteString(j)
		}
	}

	if len(so.innerJoin) > 0 {
		for _, j := range so.innerJoin {
			sb.WriteString(" INNER JOIN ")
			sb.WriteString(j)
		}
	}

	if so.where != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(so.where)
	}

	if so.group != "" {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(so.group)
	}

	if so.order != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(so.order)
	}

	if so.limits != "" {
		sb.WriteString(" LIMIT ")
		sb.WriteString(so.limits)
	}

	return sb.String(), so.vals
} // }}}

func (so *SqlOption) GetTable() string { // {{{
	return so.table
} // }}}

func (so *SqlOption) GetFields() string { // {{{
	return so.fields
} // }}}

func (so *SqlOption) GetAlias() string { // {{{
	return so.alias
} // }}}

func (so *SqlOption) GetLeftJoin() []string { // {{{
	return so.leftJoin
} // }}}

func (so *SqlOption) GetInnerJoin() []string { // {{{
	return so.innerJoin
} // }}}

func (so *SqlOption) GetIdx() string { // {{{
	return so.idx
} // }}}

func (so *SqlOption) GetGroup() string { // {{{
	return so.group
} // }}}

func (so *SqlOption) GetOrder() string { // {{{
	return so.order
} // }}}

func (so *SqlOption) GetLimits() string { // {{{
	return so.limits
} // }}}

func (so *SqlOption) GetWhere() string { // {{{
	return so.where
} // }}}

func (so *SqlOption) GetVals() []any { // {{{
	return so.vals
} // }}}

func WithTable(table string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.table = table
	}
} // }}}

func WithFields(fields string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.fields = fields
	}
} // }}}

func WithAlias(alias string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.alias = alias
	}
} // }}}

func WithLeftJoin(left_join []string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.leftJoin = left_join
	}
} // }}}

func WithInnerJoin(inner_join []string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.innerJoin = inner_join
	}
} // }}}

func WithIdx(idx string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.idx = idx
	}
} // }}}

func WithGroup(g string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.group = g
	}
} // }}}

func WithOrder(o string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.order = o
	}
} // }}}

func WithLimits(l string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.limits = l
	}
} // }}}

func WithWhere(where string, vals []any) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.where = where
		s.vals = vals
	}
} // }}}

func Expr(param string) string { // {{{
	if "" != param {
		return "#~#" + param
	}

	return ""
} // }}}

// 表达式参数
func GetExprParam(param any) string { // {{{
	if val, ok := param.(string); ok {
		if strings.HasPrefix(val, "#~#") {
			return string([]byte(val)[3:])
		}
	}

	return ""
} // }}}
