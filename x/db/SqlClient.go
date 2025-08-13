package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	p        *SqlClient //实际上没什么用，只在事务中打印调式信息时使用(因为在事务中执行explain语句会出现'busy buffer'的错误)
	id       string
}

type FnSqlOption func(*SqlOption)

type SqlOption struct {
	table     string
	fields    string
	alias     string
	leftJoin  string
	innerJoin string
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

	if so.leftJoin != "" {
		sb.WriteString(" LEFT JOIN ")
		sb.WriteString(so.leftJoin)
	}

	if so.innerJoin != "" {
		sb.WriteString(" INNER JOIN ")
		sb.WriteString(so.innerJoin)
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

func WithLeftJoin(left_join string) FnSqlOption { // {{{
	return func(s *SqlOption) {
		s.leftJoin = left_join
	}
} // }}}

func WithInnerJoin(inner_join string) FnSqlOption { // {{{
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

func (this *SqlClient) SetDB(dbt string, _db *sql.DB) error { // {{{
	this.dbType = dbt
	this.db = _db
	this.executor = &DbExecutor{this.db}

	return nil
} // }}}

func (this *SqlClient) Close() { //{{{
	this.db.Close()
} //}}}

func (this *SqlClient) Ping() error { //{{{
	return this.db.Ping()
} //}}}

func (this *SqlClient) SetDebug(open bool) { //{{{
	this.Debug = open
} //}}}

func (this *SqlClient) Type() string { //{{{
	return this.dbType
} //}}}

func (this *SqlClient) ID() string { //{{{
	if this.id == "" {
		this.id = fmt.Sprintf("%p", &this.db)
	}

	return this.id
} //}}}

func (this *SqlClient) Begin(is_readonly bool) (*SqlClient, error) { // {{{
	//tx, err := this.db.Begin()
	tx, err := this.db.BeginTx(context.Background(), &sql.TxOptions{
		ReadOnly: is_readonly,
	})

	if err != nil {
		return nil, errorHandle(fmt.Errorf("trans error:%v", err))
	}

	if this.Debug {
		if is_readonly {
			fmt.Println("Begin readonly transaction on #ID:", this.ID())
		} else {
			fmt.Println("Begin transaction on #ID:", this.ID())
		}
	}

	return &SqlClient{
		db:       this.db,
		executor: &TxExecutor{tx},
		tx:       tx,
		intx:     true,
		Debug:    this.Debug,
		p:        this,
	}, nil
} // }}}

func (this *SqlClient) Rollback() error { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Rollback()
		if err != nil {
			return errorHandle(fmt.Errorf("trans rollback error:%v", err))
		}

		if this.Debug {
			fmt.Println("Rollback transaction on #ID:", this.ID())
		}
	}

	return nil
} // }}}

func (this *SqlClient) Commit() error { // {{{
	if this.intx && nil != this.tx {
		this.intx = false
		err := this.tx.Commit()
		if err != nil {
			return errorHandle(fmt.Errorf("trans commit error:%v", err))
		}

		if this.Debug {
			fmt.Println("Commit transaction on #ID:", this.ID())
		}
	}

	return nil
} // }}}

func (this *SqlClient) Insert(table string, vals ...map[string]any) (int, error) { // {{{
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

			if fval := this.getExprParam(val); fval != "" {
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
	result, err := this.execute(sqlstr, args...)
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

func (this *SqlClient) Update(table string, vals map[string]interface{}, where string, val ...interface{}) (int, error) { // {{{
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

		if fval := this.getExprParam(val); fval != "" {
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
	result, err := this.execute(sqlstr, value...)
	if err != nil {
		return 0, err
	}

	affect, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affect), nil
} // }}}

func (this *SqlClient) Upsert(table string, vals map[string]any, ignore_fields ...string) (int, error) { // {{{
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
		if fval := this.getExprParam(val); fval != "" {
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

		if fval := this.getExprParam(val); fval != "" {
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
	result, err := this.execute(sqlstr, args...)
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

// limit <= 0 时表示删除所有符合条件的数据
func (this *SqlClient) Delete(table, order string, limit int, where string, val ...any) (int, error) { // {{{
	if "" != order {
		where += " order by " + order
	}

	if limit > 0 {
		where += " limit " + strconv.Itoa(limit)
	}

	sqlstr := "delete from " + table + " where " + where

	return this.Execute(sqlstr, val...)
} // }}}

// 表达式参数
func (this *SqlClient) getExprParam(param any) string { // {{{
	if val, ok := param.(string); ok {
		if strings.HasPrefix(val, "#~#") {
			return string([]byte(val)[3:])
		}
	}

	return ""
} // }}}

func (this *SqlClient) Execute(sqlstr string, val ...any) (int, error) { // {{{
	result, err := this.execute(sqlstr, val...)
	if err != nil {
		return 0, err
	}

	affect, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affect), nil
} // }}}

func (this *SqlClient) execute(sqlstr string, val ...interface{}) (result sql.Result, err error) { // {{{
	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	result, err = this.executor.Exec(sqlstr, val...)

	if this.Debug {
		fmt.Println(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "val": val, "#ID": this.id})
	}

	return result, errorHandle(err)
} // }}}

func (this *SqlClient) GetOne(options ...FnSqlOption) (any, error) { // {{{
	options = append(options, WithLimits("1"))
	sqlstr, vals := this.prepareSql(options)
	return this.QueryOne(sqlstr, vals...)
} // }}}

func (this *SqlClient) GetRow(options ...FnSqlOption) (map[string]any, error) { // {{{
	options = append(options, WithLimits("1"))
	sqlstr, vals := this.prepareSql(options)
	return this.QueryRow(sqlstr, vals...)
} // }}}

func (this *SqlClient) GetAll(options ...FnSqlOption) ([]map[string]any, error) { //{{{
	sqlstr, vals := this.prepareSql(options)
	return this.Query(sqlstr, vals...)
} // }}}

func (this *SqlClient) QueryOne(sqlstr string, vals ...any) (any, error) { // {{{
	var name any
	var err error

	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	err = this.executor.QueryRow(sqlstr, vals...).Scan(&name)
	if this.Debug {
		fmt.Println(map[string]any{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "vals": vals, "#ID": this.ID()})
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

func (this *SqlClient) QueryRow(sqlstr string, vals ...any) (map[string]any, error) { // {{{
	list, err := this.Query(sqlstr, vals...)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return make(map[string]any, 0), nil
} // }}}

func (this *SqlClient) Query(sqlstr string, val ...any) ([]map[string]any, error) { //{{{
	iter, err := this.QueryStream(sqlstr, val...)
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
func (this *SqlClient) QueryStream(sqlstr string, val ...any) (*RowIterator, error) { //{{{
	//分析sql,如果使用了select SQL_CALC_FOUND_ROWS, 分析语句会干扰结果，所以放在真正查询的前面
	if this.Debug {
		this.explain(sqlstr, val...)
	}

	var start_time time.Time
	if this.Debug {
		start_time = time.Now()
	}

	rows, err := this.executor.Query(sqlstr, val...)

	if this.Debug {
		fmt.Println(map[string]interface{}{"tx": this.intx, "consume": time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000, "sql": sqlstr, "val": val, "#ID": this.ID()})
	}

	if err != nil {
		return nil, errorHandle(err)
	}

	return newRowIterator(rows)
} // }}}

func (this *SqlClient) prepareSql(options []FnSqlOption) (string, []any) { //{{{
	so := &SqlOption{}
	for _, opt := range options {
		opt(so)
	}

	return so.ToSql()
} // }}}

func (this *SqlClient) explain(sqlstr string, val ...any) { //{{{
	if strings.HasPrefix(sqlstr, "select") {
		expl_results := []map[string]interface{}{}
		if this.intx {
			expl_results, _ = this.p.Query("explain "+sqlstr, val...)
		} else {
			expl_results, _ = this.Query("explain "+sqlstr, val...)
		}
		expl := &SqlExplain{this.dbType, expl_results}
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
