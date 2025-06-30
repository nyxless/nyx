package db

import (
	"database/sql"
	"fmt"
)

type RowIterator struct {
	rows     *sql.Rows
	cols     []string
	scanArgs []any
	values   []any
	closed   bool
}

// 私有方法,  由QueryStream 调用
func newRowIterator(rows *sql.Rows) (*RowIterator, error) { // {{{
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

	return &RowIterator{
		rows:     rows,
		cols:     cols,
		scanArgs: scanArgs,
		values:   values,
	}, nil
} // }}}

// 遍历每行数据，直到处理完所有行或回调返回错误
func (it *RowIterator) Foreach(fn func(map[string]any) error) error { // {{{
	if it.closed {
		return errorHandle(fmt.Errorf("iterator is already closed"))
	}

	defer it.Close()

	for it.rows.Next() {
		// 重置values
		for i := range it.values {
			it.values[i] = nil
		}

		if err := it.rows.Scan(it.scanArgs...); err != nil {
			return errorHandle(err)
		}

		row := make(map[string]any, len(it.cols))
		for i, col := range it.values {
			if col == nil {
				row[it.cols[i]] = ""
			} else if colval, ok := col.(sql.RawBytes); ok {
				row[it.cols[i]] = string(colval)
			} else {
				row[it.cols[i]] = col
			}
		}

		if err := fn(row); err != nil {
			return errorHandle(err)
		}
	}

	return errorHandle(it.rows.Err())
} // }}}

// 收集所有行数据到切片中
func (it *RowIterator) Collect() ([]map[string]any, error) { // {{{
	if it.closed {
		return nil, errorHandle(fmt.Errorf("iterator is already closed"))
	}

	defer it.Close()

	var result []map[string]any
	collectFn := func(row map[string]any) error {
		result = append(result, row)
		return nil
	}

	if err := it.Foreach(collectFn); err != nil {
		return nil, err
	}

	return result, nil
} // }}}

// 释放数据库资源
func (it *RowIterator) Close() error { // {{{
	if it.closed {
		return nil
	}

	it.closed = true
	return it.rows.Close()
} // }}}
