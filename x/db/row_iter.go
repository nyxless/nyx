package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type RowIter struct {
	rows     *sql.Rows
	cols     []string
	scanArgs []any
	values   []any
	closed   bool
}

// 私有方法,  由QueryStream 调用
func newRowIter(rows *sql.Rows) (*RowIter, error) { // {{{
	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, errorHandle(err)
	}

	values := make([]any, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	return &RowIter{
		rows:     rows,
		cols:     cols,
		scanArgs: scanArgs,
		values:   values,
	}, nil
} // }}}

// 遍历每行数据，直到处理完所有行或回调返回错误
func (it *RowIter) Foreach(fn func(map[string]any) error, limits ...int) error { // {{{
	if it.closed {
		return errorHandle(fmt.Errorf("iterator is already closed"))
	}

	limit := 0
	if len(limits) > 0 {
		limit = limits[0]
	}

	defer it.Close()

	num := 0
	for it.rows.Next() {
		if limit > 0 && num >= limit {
			return nil
		}

		// 重置values
		for i := range it.values {
			it.values[i] = nil
		}

		if err := it.rows.Scan(it.scanArgs...); err != nil {
			return errorHandle(err)
		}

		row := make(map[string]any, len(it.cols))
		for i, col := range it.values {
			colkey := it.cols[i]
			//处理返回多表字段时，字段名带前缀的问题
			li := strings.LastIndex(colkey, ".")
			if li > -1 {
				colkey = colkey[li+1:]
			}

			if colval, ok := col.(sql.RawBytes); ok {
				row[colkey] = string(colval)
			} else {
				row[colkey] = col
			}
		}

		if err := fn(row); err != nil {
			return errorHandle(err)
		}

		num++
	}

	return errorHandle(it.rows.Err())
} // }}}

// 收集所有行数据到切片中
func (it *RowIter) Collect(limits ...int) ([]map[string]any, error) { // {{{
	if it.closed {
		return nil, errorHandle(fmt.Errorf("iterator is already closed"))
	}

	defer it.Close()

	var result []map[string]any
	collectFn := func(row map[string]any) error {
		result = append(result, row)
		return nil
	}

	if err := it.Foreach(collectFn, limits...); err != nil {
		return nil, err
	}

	return result, nil
} // }}}

// 释放数据库资源
func (it *RowIter) Close() error { // {{{
	if it.closed {
		return nil
	}

	it.closed = true
	return it.rows.Close()
} // }}}
