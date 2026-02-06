package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

func NewDBClient() DBClient { // {{{
	return NewSqlClient()
} // }}}

func NewSqlClient() *SqlClient { // {{{
	return &SqlClient{}
} // }}}

type SqlClient struct {
	Debug    bool
	db       *sql.DB
	intx     bool
	tx       *sql.Tx
	executor Executor
	dbType   string
	p        *SqlClient //实际上没什么用，只在事务中打印调式信息时使用 (由于事务中执行explain语句会出现'busy buffer'的错误)
	id       string
}

func (s *SqlClient) SetDB(dbt string, _db *sql.DB) error { // {{{
	s.dbType = dbt
	s.db = _db
	s.executor = &DbExecutor{s.db}

	return nil
} // }}}

func (s *SqlClient) Close() { //{{{
	s.db.Close()
} //}}}

func (s *SqlClient) Ping(ctx context.Context) error { //{{{
	return s.db.PingContext(ctx)
} //}}}

func (s *SqlClient) SetDebug(open bool) { //{{{
	s.Debug = open
} //}}}

func (s *SqlClient) Type() string { //{{{
	return s.dbType
} //}}}

func (s *SqlClient) ID() string { //{{{
	if s.id == "" {
		id := fmt.Sprintf("%p", &s.db)
		s.id = id[max(0, len(id)-5):]
	}

	return s.id
} //}}}

func (s *SqlClient) Begin(is_readonly bool) (DBClient, error) { // {{{
	//tx, err := s.db.Begin()
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		ReadOnly: is_readonly,
	})

	if err != nil {
		return nil, errorHandle(fmt.Errorf("trans error:%v", err))
	}

	if s.Debug {
		if is_readonly {
			log.Println("Begin readonly transaction on #ID:", s.ID())
		} else {
			log.Println("Begin transaction on #ID:", s.ID())
		}
	}

	return &SqlClient{
		db:       s.db,
		executor: &TxExecutor{tx},
		tx:       tx,
		intx:     true,
		Debug:    s.Debug,
		p:        s,
	}, nil
} // }}}

func (s *SqlClient) Rollback() error { // {{{
	if s.intx && nil != s.tx {
		s.intx = false
		err := s.tx.Rollback()
		if err != nil {
			return errorHandle(fmt.Errorf("trans rollback error:%v", err))
		}

		if s.Debug {
			log.Println("Rollback transaction on #ID:", s.ID())
		}
	}

	return nil
} // }}}

func (s *SqlClient) Commit() error { // {{{
	if s.intx && nil != s.tx {
		s.intx = false
		err := s.tx.Commit()
		if err != nil {
			return errorHandle(fmt.Errorf("trans commit error:%v", err))
		}

		if s.Debug {
			log.Println("Commit transaction on #ID:", s.ID())
		}
	}

	return nil
} // }}}

func (s *SqlClient) Insert(table string, vals ...map[string]any) (int, error) { // {{{
	if len(vals) == 0 {
		return 0, nil
	}

	// 获取所有列名（假设所有map的键相同，以第一个为准）
	var columns []string
	for col := range vals[0] {
		columns = append(columns, col)
	}

	buf := bytes.NewBufferString("")

	buf.WriteString("insert into ")
	buf.WriteString(table)
	buf.WriteString(" (")
	buf.WriteString(strings.Join(columns, ", "))
	buf.WriteString(") ")
	buf.WriteString(" values ")

	// 构建占位符和值
	var placeholders []string
	var args []interface{}
	for i, row := range vals {
		// 检查每行的列是否一致
		if len(row) != len(columns) {
			return 0, fmt.Errorf("row %d has different columns count", i)
		}

		// 构建占位符
		ph := make([]string, len(columns))
		for j, col := range columns {
			val, ok := row[col]
			if !ok {
				return 0, fmt.Errorf("row %d missing column %s", i, col)
			}

			if fval := GetExprParam(val); fval != "" {
				ph[j] = fval
			} else {
				ph[j] = "?"
				args = append(args, val)
			}
		}
		placeholders = append(placeholders, "("+strings.Join(ph, ", ")+")")
	}

	buf.WriteString(strings.Join(placeholders, ", "))

	sqlstr := buf.String()
	result, err := s.Exec(sqlstr, args...)
	if err != nil {
		return 0, err
	}

	lastid, err := result.LastInsertId()
	if err != nil {
		if err.Error() == "LastInsertId is not supported by this driver" {
			return 0, nil
		}

		return 0, err
	}

	return int(lastid), nil
} // }}}

func (s *SqlClient) Update(table string, vals map[string]interface{}, where string, val ...interface{}) (int, error) { // {{{
	buf := bytes.NewBufferString("update ")

	buf.WriteString(table)
	buf.WriteString(" set ")

	var value []interface{}
	i := 0
	for col, val := range vals {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(col)
		buf.WriteString("=")

		if fval := GetExprParam(val); fval != "" {
			buf.WriteString(fval)
		} else {
			buf.WriteString("?")
			value = append(value, val)
		}

		i++
	}

	buf.WriteString(" where ")
	buf.WriteString(where)
	sqlstr := buf.String()

	value = append(value, val...)
	result, err := s.Exec(sqlstr, value...)
	if err != nil {
		return 0, err
	}

	affect, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affect), nil
} // }}}

func (s *SqlClient) Upsert(table string, vals map[string]any, ignore_fields ...string) (int, error) { // {{{
	if len(vals) == 0 {
		return 0, nil
	}

	// 获取列名和值
	var columns []string
	var placeholders []string
	var args []interface{}
	var updateParts []string

	// 插入
	for col, val := range vals {
		columns = append(columns, col)
		if fval := GetExprParam(val); fval != "" {
			placeholders = append(placeholders, fval)
		} else {
			placeholders = append(placeholders, "?")
			args = append(args, val)
		}
	}

	// 更新（排除忽略字段）
	for col, val := range vals {
		if contains(ignore_fields, col) {
			continue
		}

		if fval := GetExprParam(val); fval != "" {
			updateParts = append(updateParts, col+" = "+fval)
		} else {
			updateParts = append(updateParts, col+" = ?")
			args = append(args, val)
		}
	}

	buf := bytes.NewBufferString("")
	buf.WriteString("INSERT INTO ")
	buf.WriteString(table)
	buf.WriteString(" (")
	buf.WriteString(strings.Join(columns, ", "))
	buf.WriteString(") VALUES (")
	buf.WriteString(strings.Join(placeholders, ", "))
	buf.WriteString(") ON DUPLICATE KEY UPDATE ")
	buf.WriteString(strings.Join(updateParts, ", "))

	sqlstr := buf.String()
	result, err := s.Exec(sqlstr, args...)
	if err != nil {
		return 0, err
	}

	lastid, err := result.LastInsertId()
	if err != nil {
		if err.Error() == "LastInsertId is not supported by this driver" {
			return 0, nil
		}
		return 0, err
	}

	return int(lastid), nil
} // }}}

func (s *SqlClient) Delete(options ...FnSqlOption) (int, error) { // {{{
	var sb strings.Builder

	so := s.parseOptions(options)

	sb.WriteString("DELETE")

	if so.table != "" {
		sb.WriteString(" FROM ")
		sb.WriteString(so.table)
	}

	if so.where != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(so.where)
	}

	if so.order != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(so.order)
	}

	if so.limits != "" {
		sb.WriteString(" LIMIT ")
		sb.WriteString(so.limits)
	}

	return s.Execute(sb.String(), so.vals...)
} // }}}

func (s *SqlClient) Execute(sqlstr string, val ...any) (int, error) { // {{{
	result, err := s.Exec(sqlstr, val...)
	if err != nil {
		return 0, err
	}

	affect, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affect), nil
} // }}}

func (s *SqlClient) Exec(sqlstr string, val ...interface{}) (result sql.Result, err error) { // {{{
	var start_time time.Time
	if s.Debug {
		start_time = time.Now()
	}

	result, err = s.executor.Exec(sqlstr, val...)

	if s.Debug {
		log.Println(map[string]interface{}{"tx": s.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "val": val, "#ID": s.id})
	}

	return result, errorHandle(err)
} // }}}

func (s *SqlClient) GetOne(options ...FnSqlOption) (any, error) { // {{{
	options = append(options, WithLimits("1"))
	sqlstr, vals := s.prepareSql(options)
	return s.QueryOne(sqlstr, vals...)
} // }}}

func (s *SqlClient) GetRow(options ...FnSqlOption) (map[string]any, error) { // {{{
	options = append(options, WithLimits("1"))
	sqlstr, vals := s.prepareSql(options)
	return s.QueryRow(sqlstr, vals...)
} // }}}

func (s *SqlClient) GetAll(options ...FnSqlOption) ([]map[string]any, error) { //{{{
	sqlstr, vals := s.prepareSql(options)
	return s.Query(sqlstr, vals...)
} // }}}

func (s *SqlClient) QueryOne(sqlstr string, vals ...any) (any, error) { // {{{
	var name any
	var err error

	var start_time time.Time
	if s.Debug {
		start_time = time.Now()
	}

	err = s.executor.QueryRow(sqlstr, vals...).Scan(&name)
	if s.Debug {
		log.Println(map[string]any{"tx": s.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "vals": vals, "#ID": s.ID()})
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, errorHandle(err)
		}
	}

	return name, nil
} // }}}

func (s *SqlClient) QueryRow(sqlstr string, vals ...any) (map[string]any, error) { // {{{
	iter, err := s.QueryStream(sqlstr, vals...)
	if err != nil {
		return nil, errorHandle(err)
	}

	list, err := iter.Collect(1)
	if err != nil {
		return nil, errorHandle(err)
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return make(map[string]any, 0), nil
} // }}}

func (s *SqlClient) Query(sqlstr string, val ...any) ([]map[string]any, error) { //{{{
	iter, err := s.QueryStream(sqlstr, val...)
	if err != nil {
		return nil, errorHandle(err)
	}

	data, err := iter.Collect()
	if err != nil {
		return nil, errorHandle(err)
	}

	return data, nil
} // }}}

// 返回迭代器
func (s *SqlClient) QueryStream(sqlstr string, val ...any) (*RowIter, error) { //{{{
	//分析sql,如果使用了select SQL_CALC_FOUND_ROWS, 分析语句会干扰结果，所以放在真正查询的前面
	if s.Debug {
		s.explain(sqlstr, val...)
	}

	var start_time time.Time
	if s.Debug {
		start_time = time.Now()
	}

	rows, err := s.executor.Query(sqlstr, val...)

	if s.Debug {
		log.Println(map[string]interface{}{"tx": s.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "val": val, "#ID": s.ID()})
	}

	if err != nil {
		return nil, errorHandle(err)
	}

	return newRowIter(rows)
} // }}}

func (s *SqlClient) parseOptions(options []FnSqlOption) *SqlOption { //{{{
	so := &SqlOption{}
	for _, opt := range options {
		opt(so)
	}

	return so
} // }}}

func (s *SqlClient) prepareSql(options []FnSqlOption) (string, []any) { //{{{
	return s.parseOptions(options).ToSql()
} // }}}

func (s *SqlClient) explain(sqlstr string, val ...any) { //{{{
	if strings.HasPrefix(sqlstr, "select") {
		expl_results := []map[string]interface{}{}
		if s.intx {
			expl_results, _ = s.p.Query("explain "+sqlstr, val...)
		} else {
			expl_results, _ = s.Query("explain "+sqlstr, val...)
		}
		expl := &SqlExplain{s.dbType, expl_results}
		expl.DrawConsole()
	}
} // }}}

// 检查字符串是否在切片中
func contains(slice []string, item string) bool { // {{{
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
} // }}}
