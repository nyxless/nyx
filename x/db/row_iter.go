package db

import (
	"bytes"
	"database/sql"
	"fmt"
)

type RowIter struct {
	rows     *sql.Rows
	cols     []string
	scanArgs []any
	values   []any
	closed   bool
	useBytes bool //  是否保留 []byte, sql.RawBytes  字段值类型, 默认转为 string
}

// 私有方法,  由QueryStream 调用
func newRowIter(rows *sql.Rows, use_bytes bool) (*RowIter, error) { // {{{
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
		useBytes: use_bytes,
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
		for i := range it.values {
			row[it.cols[i]] = parseValue(it.values[i], it.useBytes)
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

func parseValue(val any, use_bytes bool) any { // {{{
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case sql.RawBytes:
		if use_bytes {
			return bytes.Clone(v)
		}
		return string(v)
	case []byte:
		if use_bytes {
			return bytes.Clone(v)
		}
		return string(v)
	case sql.NullString:
		if v.Valid {
			return v.String
		}
		return nil
	case sql.NullInt16:
		if v.Valid {
			return v.Int16
		}
		return nil
	case sql.NullInt32:
		if v.Valid {
			return v.Int32
		}
		return nil
	case sql.NullInt64:
		if v.Valid {
			return v.Int64
		}
		return nil
	case sql.NullFloat64:
		if v.Valid {
			return v.Float64
		}
		return nil
	case sql.NullBool:
		if v.Valid {
			return v.Bool
		}
		return nil
	case sql.NullTime:
		if v.Valid {
			return v.Time
		}
		return nil
	case sql.NullByte:
		if v.Valid {
			return v.Byte
		}
		return nil
	default:
		return v
	}

	return val
} // }}}
