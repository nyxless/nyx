package dao

import (
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/db"
	"strconv"
	"strings"
	"time"
)

// GetOne、GetRecord、GetRecordBy 时无记录会返回 sql.ErrNoRows
var ErrNoRows = sql.ErrNoRows

type Dao struct {
	DBWriter, DBReader   db.DBClient
	table                string
	primary              string
	defaultFields        string   //默认字段,逗号分隔
	fields               []string //通过setFields方法指定的字段,逗号分隔,只能通过getFields使用一次
	countField           string   //getCount方法使用的字段
	index                string   //查询使用的索引
	limit                string
	autoOrder            bool //是否自动排序(默认按自动主键倒序排序)
	order                []string
	group                string
	filter               [][]any //过滤条件
	forceMaster          bool    //强制使用主库读，只能通过useMaster 使用一次
	ctx                  context.Context
	alias                string //表别名
	leftJoin             []*Dao
	innerJoin            []*Dao
	onPairs              []*onPair
	cnt                  *int //存放查询总数的变量指针
	useCache             bool //使用缓存
	cacheTtl             int  //缓存时长，单位秒
	cacheRefreshInterval int  //缓存自动刷新间隔，单位秒
	cacheCallbackFn      CacheCallbackFn
	hit                  bool      // 是否命中缓存
	cacheTime            time.Time // 临时存放缓存时间
	useBytes             bool      // 是否保留 []byte, sql.RawBytes  字段值类型, 默认转为 string
}

// 缓存更新成功后回调函数, any 参数类型和当前调用的查询方法返回数据类型一致, 如 GetRecord: map[string]any, GetRecords: []map[string]any
type CacheCallbackFn func(any) error

type onPair struct {
	Left    string
	Right   string
	Compare string
}

// 逻辑条件组合
type Cond struct {
	op    string // "AND", "OR", "NOT"
	conds []any  // 只接受 *Cond 和 map[string]any
}

// 逻辑组合函数
func And(conds ...any) *Cond {
	return &Cond{op: "AND", conds: conds}
}

func Or(conds ...any) *Cond {
	return &Cond{op: "OR", conds: conds}
}

func Not(cond any) *Cond {
	return &Cond{op: "NOT", conds: []any{cond}}
}

func (d *Dao) Init(conf_name ...string) { //{{{
	master_conf_name := "db_master"
	slave_conf_name := "db_slave"

	if len(conf_name) > 0 {
		master_conf_name = conf_name[0]
	}

	master_conf := x.Conf.GetMap(master_conf_name)
	if 0 == len(master_conf) {
		x.Panic("db资源不存在:" + master_conf_name)
	}

	slave_conf := master_conf

	if len(conf_name) > 1 {
		slave_conf_name = conf_name[1]
	}

	slave_confs := x.Conf.GetMapSlice(slave_conf_name)

	if len(slave_confs) > 0 {
		idx := x.RandIntn(len(slave_confs))
		slave_conf = slave_confs[idx]
	}

	if 0 == len(slave_conf) {
		fmt.Printf("从库资源不存在[ %s ], 使用主库 [ %s ]\n", slave_conf_name, master_conf_name)
	}

	var err error

	d.defaultFields = "*"

	d.DBWriter, err = x.DB.Get(master_conf)
	if err != nil {
		x.Panic(err)
	}

	d.DBReader, err = x.DB.Get(slave_conf)
	if err != nil {
		x.Panic(err)
	}

	if d.DBWriter.Type() == "mysql" {
		d.autoOrder = true
	}

	gob.Register(&CacheData{})
	gob.Register(time.Time{})
	gob.Register(map[string]interface{}{})
	gob.Register([]map[string]interface{}{})
} // }}}

func (d *Dao) InitTx(tx db.DBClient) { //使用事务{{{
	d.defaultFields = "*"
	d.autoOrder = true
	d.DBWriter = tx
	d.DBReader = tx
} // }}}

func (d *Dao) WithContext(ctx context.Context) *Dao {
	d.ctx = ctx

	return d
}

func (d *Dao) SetTable(table string) {
	d.table = table
}

func (d *Dao) GetTable() string {
	return d.table
}

func (d *Dao) SetPrimary(field string) {
	d.primary = field
}

func (d *Dao) GetPrimary() string {
	return d.primary
}

func (d *Dao) SetCountField(field string) *Dao { // {{{
	d.countField = field
	return d
} // }}}

func (d *Dao) GetCountField() string { // {{{
	if "" != d.countField {
		field := d.countField
		d.countField = ""

		if d.alias != "" && strings.Index(field, ".") == -1 {
			field = d.alias + "." + field
		}

		return field
	}

	return "1"
} // }}}

func (d *Dao) SetDefaultFields(fields ...string) *Dao { // {{{
	d.defaultFields = strings.Join(fields, ",")
	return d
} // }}}

// 可在读方法前使用，且仅对本次查询起作用，如: NewDAOUser().SetFields("uid").GetRecord(uid)
func (d *Dao) SetFields(fields ...string) *Dao {
	d.fields = append(d.fields, fields...)
	return d
}

func (d *Dao) GetFields() string { // {{{
	fields := d.defaultFields
	if len(d.fields) > 0 {
		fields = strings.Join(d.fields, ",")
		d.fields = nil
	}

	return fields
} // }}}

func (d *Dao) UseIndex(idx string) *Dao {
	d.index = idx
	return d
}

// 支持 GetRecords 方法, 返回符合条件的总数, 赋值给参数 cnt
func (d *Dao) WithCount(cnt *int) *Dao {
	d.cnt = cnt

	return d
}

func (d *Dao) getIndex() string { // {{{
	idx := d.index
	d.index = ""
	return idx
} // }}}

// 强制使用主库
func (d *Dao) UseMaster(flag ...bool) *Dao { // {{{
	use := true
	if len(flag) > 0 && !flag[0] {
		use = false
	}

	d.forceMaster = use
	return d
} // }}}

// 返回缓存时间
func (d *Dao) GetCacheTime() (time.Time, bool) { // {{{
	cacheTime := d.cacheTime
	hit := d.hit
	d.cacheTime = time.Time{}
	d.hit = false

	return cacheTime, hit
} // }}}

func (d *Dao) WithCache(ttl int, callbackFns ...CacheCallbackFn) *Dao { // {{{
	d.useCache = true
	d.cacheTtl = ttl

	if len(callbackFns) > 0 {
		d.cacheCallbackFn = callbackFns[0]
	}

	return d
} // }}}

func (d *Dao) WithRefreshCache(refreshInterval int, callbackFns ...CacheCallbackFn) *Dao { // {{{
	d.useCache = true
	d.cacheRefreshInterval = refreshInterval

	if len(callbackFns) > 0 {
		d.cacheCallbackFn = callbackFns[0]
	}

	return d
} // }}}

func (d *Dao) getUseCache() (bool, int, int, CacheCallbackFn) { // {{{
	use_cache := d.useCache
	ttl := d.cacheTtl
	refreshInterval := d.cacheRefreshInterval
	callbackFn := d.cacheCallbackFn

	d.useCache = false
	d.cacheTtl = 0
	d.cacheRefreshInterval = 0
	d.cacheCallbackFn = nil

	return use_cache, ttl, refreshInterval, callbackFn
} // }}}

func (d *Dao) SetAutoOrder(flag ...bool) *Dao { // {{{
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	d.autoOrder = use
	return d
} // }}}

func (d *Dao) Order(order ...string) *Dao { // {{{
	d.order = append(d.order, order...)
	d.autoOrder = false
	return d
} // }}}

func (d *Dao) getOrder(use_auto_order bool) string { // {{{
	order := ""
	if len(d.order) > 0 {
		order = strings.Join(d.order, ",")
		d.order = nil
	} else if use_auto_order && d.autoOrder {
		order = d.GetPrimary() + " desc"
	}

	d.autoOrder = true
	return order
} // }}}

func (d *Dao) Group(group ...string) *Dao { // {{{
	d.group = strings.Join(group, ",")
	return d
} // }}}

func (d *Dao) getGroup() string { // {{{
	group := d.group
	d.group = ""

	return group
} // }}}

func (d *Dao) Limit(limit int, limits ...int) *Dao { // {{{
	d.limit = strconv.Itoa(limit)

	if len(limits) > 0 {
		d.limit = d.limit + "," + strconv.Itoa(limits[0])
	}

	return d
} // }}}

func (d *Dao) WithBytes(flag ...bool) *Dao { // {{{
	use := true
	if len(flag) > 0 && !flag[0] {
		use = false
	}

	d.useBytes = use
	return d
} // }}}

func (d *Dao) getUseBytes() bool { // {{{
	use_bytes := d.useBytes
	d.useBytes = false

	return use_bytes
} // }}}

func (d *Dao) getLimit() string { // {{{
	limit := d.limit
	d.limit = ""

	return limit
} // }}}

func (d *Dao) Alias(alias string) *Dao { // {{{
	d.alias = alias
	return d
} // }}}

func (d *Dao) LeftJoin(left_join ...*Dao) *Dao { // {{{
	d.leftJoin = append(d.leftJoin, left_join...)
	if d.alias == "" {
		d.alias = "t"
	}

	return d
} // }}}

func (d *Dao) InnerJoin(inner_join ...*Dao) *Dao { // {{{
	d.innerJoin = append(d.innerJoin, inner_join...)
	if d.alias == "" {
		d.alias = "t"
	}

	return d
} // }}}

// alias for InnerJoin
func (d *Dao) Join(inner_join ...*Dao) *Dao { // {{{
	return d.InnerJoin(inner_join...)
} // }}}

func (d *Dao) On(left_field string, right_fields ...string) *Dao { // {{{
	return d.on("=", left_field, right_fields...)
} // }}}

func (d *Dao) NotOn(left_field string, right_fields ...string) *Dao { // {{{
	return d.on("!=", left_field, right_fields...)
} // }}}

// compare: 比较符号
func (d *Dao) CompareOn(compare, left_field string, right_fields ...string) *Dao { // {{{
	return d.on(compare, left_field, right_fields...)
} // }}}

func (d *Dao) on(compare, left_field string, right_fields ...string) *Dao { // {{{
	right_field := left_field
	if len(right_fields) > 0 {
		right_field = right_fields[0]
	}

	d.onPairs = append(d.onPairs, &onPair{left_field, right_field, compare})
	return d
} // }}}

func (d *Dao) GetDBReader() db.DBClient { // {{{
	if d.forceMaster {
		d.forceMaster = false

		return d.DBWriter
	}

	return d.DBReader
} // }}}

// 解析查询参数，返回WHERE子句和参数值列表
// 例1:parseParams("x=? and y=?", 1, 2)
// 例2:parseParams("x=? and y=?", []any{1,2}) 等价于 parseParams("a=? and b=?", 1, 2) //若第二个参数非[]any(如[]int、[]string), 可先使用 AsSlice 进行转换
// 例3:parseParams(map[string]any{"a":1,"b":2}) 等价于 parseParams("a=? and b=?", 1, 2)
// 例4:parseParams(map[string]any{"a":1,"b":[]any{2, 3}}) 等价于 parseParams("a=? and b in ('2','3')", 1)
func (d *Dao) parseParams(params ...any) (string, []any) { // {{{
	if len(params) == 0 || params[0] == nil {
		return "", nil
	}

	// 第一个参数是 string 且包含 ?，作为 SQL 模板
	if sql, ok := params[0].(string); ok && strings.Contains(sql, "?") {
		if len(params) == 1 {
			return sql, nil
		}

		// 若第二个参数是[]any，使用它作为值列表
		if slice, ok := params[1].([]any); ok && len(params) == 2 {
			return sql, slice
		}
		return sql, params[1:]
	}

	// 多参数模式，每个参数都是一个条件，用 AND 连接
	var allParts []string
	var allVals []any

	for _, p := range params {
		var part string
		var vals []any

		switch v := p.(type) {
		case *Cond:
			part, vals = parseCond(v, d.alias)
		case map[string]any:
			part, vals = parseMap(v, d.alias)
		case string:
			part = v
			vals = nil
		default:
			// 其他类型转为字符串条件
			part = fmt.Sprint(v)
			vals = nil
		}

		if part != "" {
			allParts = append(allParts, part)
			allVals = append(allVals, vals...)
		}
	}

	if len(allParts) == 0 {
		return "", nil
	}

	return strings.Join(allParts, " AND "), allVals
} // }}}

// 解析逻辑组合
func parseCond(c *Cond, alias string) (string, []any) { // {{{
	if len(c.conds) == 0 {
		return "", nil
	}

	var parts []string
	var vals []any

	for _, child := range c.conds {
		var sql string
		var val []any

		switch ch := child.(type) {
		case *Cond:
			sql, val = parseCond(ch, alias)
		case map[string]any:
			sql, val = parseMap(ch, alias)
		default:
			continue
		}

		if sql != "" {
			parts = append(parts, sql)
			vals = append(vals, val...)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	switch c.op {
	case "NOT":
		return "NOT (" + parts[0] + ")", vals
	case "AND", "OR":
		return "(" + strings.Join(parts, " "+c.op+" ") + ")", vals
	default:
		return parts[0], vals
	}
} // }}}

// 解析 map 条件（支持后缀运算符）
func parseMap(m map[string]any, defAlias string) (string, []any) { // {{{
	if len(m) == 0 {
		return "", nil
	}

	var parts []string
	var vals []any

	for key, val := range m {
		var alias string
		if dot := strings.Index(key, "."); dot > 0 {
			alias = key[:dot+1]
			key = key[dot+1:]
		} else if defAlias != "" {
			alias = defAlias + "."
		}

		// 解析后缀运算符 (field:op)
		if idx := strings.LastIndex(key, ":"); idx > 0 {
			field := key[:idx]
			op := key[idx+1:]

			switch op {
			case "gt":
				parts = append(parts, fmt.Sprintf("%s`%s` > ?", alias, field))
				vals = append(vals, val)
			case "gte":
				parts = append(parts, fmt.Sprintf("%s`%s` >= ?", alias, field))
				vals = append(vals, val)
			case "lt":
				parts = append(parts, fmt.Sprintf("%s`%s` < ?", alias, field))
				vals = append(vals, val)
			case "lte":
				parts = append(parts, fmt.Sprintf("%s`%s` <= ?", alias, field))
				vals = append(vals, val)
			case "ne":
				parts = append(parts, fmt.Sprintf("%s`%s` != ?", alias, field))
				vals = append(vals, val)
			case "like":
				parts = append(parts, fmt.Sprintf("%s`%s` LIKE ?", alias, field))
				vals = append(vals, val)
			case "notlike":
				parts = append(parts, fmt.Sprintf("%s`%s` NOT LIKE ?", alias, field))
				vals = append(vals, val)
			case "in":
				sql, vs := buildIn(alias, field, "IN", val)
				parts = append(parts, sql)
				vals = append(vals, vs...)
			case "notin":
				sql, vs := buildIn(alias, field, "NOT IN", val)
				parts = append(parts, sql)
				vals = append(vals, vs...)
			case "btw":
				sql, vs := buildBetween(alias, field, val)
				parts = append(parts, sql)
				vals = append(vals, vs...)
			case "null":
				parts = append(parts, fmt.Sprintf("%s`%s` IS NULL", alias, field))
			case "notnull":
				parts = append(parts, fmt.Sprintf("%s`%s` IS NOT NULL", alias, field))
			case "expr":
				parts = append(parts, fmt.Sprintf("%s`%s` = %v", alias, field, val))
			default:
				// 默认等于
				parts = append(parts, fmt.Sprintf("%s`%s` = ?", alias, field))
				vals = append(vals, val)
			}
		} else {
			// 无运算符，默认等于
			parts = append(parts, fmt.Sprintf("%s`%s` = ?", alias, key))
			vals = append(vals, val)
		}
	}

	return strings.Join(parts, " AND "), vals
} // }}}

// 构建 IN 条件
func buildIn(alias, field, op string, val any) (string, []any) { // {{{
	v := x.AsSlice(val)

	if len(v) == 0 {
		if op == "IN" {
			return "1=0", nil
		}
		return "1=1", nil
	}

	places := make([]string, len(v))
	for i := 0; i < len(v); i++ {
		places[i] = "?"
	}

	return fmt.Sprintf("%s`%s` %s (%s)", alias, field, op, strings.Join(places, ",")), v
} // }}}

// buildBetween 构建 BETWEEN 条件
func buildBetween(alias, field string, val any) (string, []any) { // {{{
	v := x.AsSlice(val)
	if len(v) != 2 {
		return "1=0", nil
	}

	return fmt.Sprintf("%s`%s` BETWEEN ? AND ?", alias, field), v
} // }}}

func (d *Dao) Where(params ...any) *Dao { //{{{
	return d.SetFilter(params...)
} // }}}

func (d *Dao) SetFilter(params ...any) *Dao { //{{{
	d.filter = append(d.filter, params)

	return d
} //}}}

func (d *Dao) getFilter() (string, []any) { // {{{
	var where string
	var values []any

	for i := range d.filter {
		w, v := d.parseParams(d.filter[i]...)

		if w != "" {
			if where != "" {
				where += " AND " + w
			} else {
				where = w
			}

			if len(v) > 0 {
				values = append(values, v...)
			}
		}
	}

	return where, values
} // }}}

// 在主库执行 sql 操作
func (d *Dao) Execute(sql string, params ...any) (int, error) { //{{{
	return d.DBWriter.Execute(sql, params...)
} // }}}

// 在从库执行 sql 查询单字段, 返回 any
func (d *Dao) QueryOne(sql string, params ...any) (any, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithSql(sql, params),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		val, err := d.GetDBReader().QueryOne(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, val, err

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res, nil
} // }}}

// 在从库执行 sql 查询单行, 返回 Map
func (d *Dao) QueryRow(sql string, params ...any) (map[string]any, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithSql(sql, params),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		row, err := d.GetDBReader().QueryRow(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, row, err

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[string]any), nil
} // }}}

// 在从库执行 sql 查询, 返回 MapSlice
func (d *Dao) Query(sql string, params ...any) ([]map[string]any, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithSql(sql, params),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		data, err := d.GetDBReader().Query(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, data, err

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.([]map[string]any), nil
} // }}}

// 返回迭代器, 不支持 cache
func (d *Dao) QueryStream(sql string, params ...any) (*db.RowIter, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithSql(sql, params),
		db.WithBytes(d.getUseBytes()),
	}
	return d.GetDBReader().QueryStream(sqlOptions...)
} // }}}

// 插入新记录, 支持批量
func (d *Dao) AddRecord(records ...map[string]any) (int, error) { //{{{
	return d.DBWriter.Insert(d.table, records...)
} // }}}

// 按主键更新记录, id 参数为主键值
func (d *Dao) SetRecord(record map[string]any, id any) (int, error) { //{{{
	delete(record, d.primary)
	return d.DBWriter.Update(d.table, record, d.primary+"=?", id)
} // }}}

// 按条件更新记录
func (d *Dao) SetRecordBy(record map[string]any, where string, params ...any) (int, error) { //{{{
	return d.DBWriter.Update(d.table, record, where, params...)
} // }}}

// upsert 操作
func (d *Dao) ResetRecord(record map[string]any) (int, error) { //{{{
	return d.DBWriter.Upsert(d.table, record, d.primary)
} // }}}

// 按主键查询记录
func (d *Dao) GetRecord(id any) (map[string]any, error) { //{{{
	var primary string
	if d.alias != "" {
		primary = d.alias + "." + d.primary
	} else {
		primary = d.primary
	}

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithWhere(primary+"=?", []any{id}),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		row, err := d.GetDBReader().GetRow(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, row, err

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[string]any), nil
} // }}}

func (d *Dao) parseJoin(joinons []*Dao) []string { // {{{
	if len(joinons) == 0 {
		return nil
	}

	var joins []string
	for i, v := range joinons {
		var join string

		tbl := v.GetTable()
		al := v.alias
		if al == "" {
			al = "t" + strconv.Itoa(i)
			v.Alias(al)
		}
		join += tbl + " " + al

		for i, p := range v.onPairs {
			if i == 0 {
				join += " ON "
			} else {
				join += " AND "
			}

			on := "="
			if p.Compare != "" {
				on = p.Compare
			}

			join += d.alias + "." + p.Left + on + al + "." + p.Right
		}

		joins = append(joins, join)

		if len(v.fields) > 0 {
			d.SetFields(db.FillAlias(al, strings.Join(v.fields, ",")))
		}

		if len(v.order) > 0 {
			d.Order(db.FillAlias(al, strings.Join(v.order, ",")))
		}

		if where, vals := v.getFilter(); where != "" {
			d.SetFilter(where, vals)
		}
	}

	return joins
} // }}}

// 按主键删除数据
func (d *Dao) DelRecord(id any) (int, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithWhere(d.primary+"=?", []any{id}),
		db.WithLimits("1"),
	}

	return d.DBWriter.Delete(sqlOptions...)
} // }}}

// 删除符合条件的数据 (一条)
func (d *Dao) DelRecordBy(params ...any) (int, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithLimits("1"),
	}

	return d.DBWriter.Delete(sqlOptions...)
} // }}}

// 删除所有符合条件的数据 (Is Dangerous!)
func (d *Dao) DelRecords(params ...any) (int, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithWhere(d.getFilter()),
	}

	return d.DBWriter.Delete(sqlOptions...)
} // }}}

func (d *Dao) GetOne(field string, params ...any) (any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {
		res, err := d.GetDBReader().GetOne(sqlOptions...)
		return 0, res, err
	}, sqlOptions)

	return res, err
} // }}}

// alias for GetOne
func (d *Dao) GetValue(field string, params ...any) (any, error) { //{{{
	return d.GetOne(field, params...)
} // }}}

func (d *Dao) GetValues(field string, params ...any) ([]any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		list, err := d.GetDBReader().GetAll(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		if len(list) > 0 {
			for k, _ := range list[0] {
				field = k
				break
			}
		}

		return 0, x.ArrayColumn(list, field), nil

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.([]any), err
} // }}}

func (d *Dao) GetUniqueValues(field string, params ...any) ([]any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		list, err := d.GetDBReader().GetAll(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		if len(list) > 0 {
			for k, _ := range list[0] {
				field = k
				break
			}
		}

		return 0, x.ArrayColumnUnique(list, field), nil

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.([]any), err
} // }}}

func (d *Dao) GetValuesMap(keyfield, valfield string, params ...any) (map[any]any, error) { //{{{
	d.SetFilter(params...)

	field := d.GetFields()
	if field == d.defaultFields {
		field = keyfield + "," + valfield
	}

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		list, err := d.GetDBReader().GetAll(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, x.ArrayColumnMap(list, keyfield, valfield), nil

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[any]any), err
} // }}}

func (d *Dao) GetGroupMap(keyfield string, params ...any) (map[any]map[string]any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		list, err := d.GetDBReader().GetAll(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, x.ArrayGroupMap(list, keyfield), nil

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[any]map[string]any), err
} // }}}

func (d *Dao) GetGroupMaps(keyfield string, params ...any) (map[any][]map[string]any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder(false)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		list, err := d.GetDBReader().GetAll(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, x.ArrayGroupMaps(list, keyfield), nil

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[any][]map[string]any), err
} // }}}

func (d *Dao) GetCount(params ...any) (int, error) { //{{{
	d.SetFilter(params...)

	field := "count(" + d.GetCountField() + ") as total"

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithWhere(d.getFilter()),
	}

	res, err := d.getCache(func() (int, any, error) {

		count, err := d.GetDBReader().GetOne(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}
		return 0, count, nil

	}, sqlOptions)

	return x.AsInt(res), err
} // }}}

func (d *Dao) Exists(id any) (bool, error) { //{{{
	one, err := d.GetOne(d.primary, d.primary+"=?", id)
	if err != nil {
		return false, err
	}

	return one != nil, nil
} // }}}

func (d *Dao) ExistsBy(params ...any) (bool, error) { //{{{
	one, err := d.GetOne(d.primary, params...)
	if err != nil {
		return false, err
	}

	return one != nil, nil
} // }}}

func (d *Dao) GetRecordBy(params ...any) (map[string]any, error) { //{{{
	d.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(d.alias),
		db.WithLeftJoin(d.parseJoin(d.leftJoin)),
		db.WithInnerJoin(d.parseJoin(d.innerJoin)),
		db.WithWhere(d.getFilter()),
		db.WithBytes(d.getUseBytes()),
	}

	res, err := d.getCache(func() (int, any, error) {

		row, err := d.GetDBReader().GetRow(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}

		return 0, row, err

	}, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.(map[string]any), err

} // }}}

func (d *Dao) GetRecords(params ...any) ([]map[string]any, error) { //{{{
	d.SetFilter(params...)

	left_join := d.parseJoin(d.leftJoin)
	inner_join := d.parseJoin(d.innerJoin)

	idx := d.getIndex()
	group := d.getGroup()
	where, values := d.getFilter()

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(d.alias),
		db.WithIdx(idx),
		db.WithGroup(group),
		db.WithOrder(d.getOrder(true)),
		db.WithLimits(d.getLimit()),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
		db.WithWhere(where, values),
		db.WithBytes(d.getUseBytes()),
	}

	getRecordsFn := func() (int, any, error) {
		res, err := d.GetDBReader().GetAll(sqlOptions...)
		return 0, res, err
	}

	// 查询列表 + 总数, 总数赋值到指定变量cnt
	// *mysql 支持 FOUND_ROWS(), 但大数据下可能存在性能问题，因此使用更兼容的方案
	if d.cnt != nil {
		getRecordsFn = func() (int, any, error) {
			var count any
			var list []map[string]any
			var err error

			db_reader := d.GetDBReader()
			list, err = db_reader.GetAll(sqlOptions...)

			if err != nil {
				return 0, nil, err
			}

			count, err = db_reader.GetOne(db.WithTable(d.table), db.WithAlias(d.alias), db.WithLeftJoin(left_join), db.WithInnerJoin(inner_join), db.WithFields("count(1) as total"), db.WithIdx(idx), db.WithGroup(group), db.WithWhere(where, values))
			if err != nil {
				return 0, nil, err
			}

			return x.AsInt(count), list, nil
		}
	}

	res, err := d.getCache(getRecordsFn, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.([]map[string]any), err
} // }}}

func (d *Dao) getCache(fn func() (int, any, error), opts []db.FnSqlOption) (res any, err error) { //{{{
	use_cache, ttl, refreshInterval, callbackFn := d.getUseCache()

	var num int
	if use_cache && x.LocalCache != nil {
		cache_data, hit, err := d.getFromCache(fn, opts, ttl, refreshInterval, callbackFn)
		if err != nil {
			return nil, err
		}
		num = cache_data.Num
		res = cache_data.Res

		d.hit = hit
		d.cacheTime = cache_data.CacheTime
	} else {
		num, res, err = fn()
	}

	if d.cnt != nil {
		*d.cnt = num
		d.cnt = nil
	}

	return res, err
} // }}}

func (d *Dao) getFromCache(fn func() (int, any, error), opts []db.FnSqlOption, ttl, refreshInterval int, callbackFn CacheCallbackFn) (*CacheData, bool, error) { // {{{
	key := d.getCacheKey(opts)
	cacheFn := func() ([]byte, bool, error) {
		num, res, err := fn()
		if nil != err {
			return nil, false, err
		}

		cache_data := &CacheData{num, res, x.NowTime()}
		data, err := x.GobEncode(cache_data)
		if nil != err {
			return nil, false, err
		}

		return data, true, nil
	}

	t := ttl
	cacheGetFn := x.LocalCache.GetOrSetFn
	if refreshInterval > 0 {
		t = refreshInterval
		cacheGetFn = x.LocalCache.GetOrRefreshFn
	}

	var baseCacheCallbackFn func([]byte) error
	if callbackFn != nil {
		baseCacheCallbackFn = func(data []byte) error {
			var cache_data *CacheData
			err := x.GobDecode(data, &cache_data)
			if err != nil {
				return err
			}

			return callbackFn(cache_data.Res)
		}
	}

	got, hit, err := cacheGetFn(key, cacheFn, t, baseCacheCallbackFn)
	if err != nil {
		return nil, false, err
	}

	var cache_data *CacheData
	err = x.GobDecode(got, &cache_data)
	if err != nil {
		return nil, false, err
	}

	return cache_data, hit, nil
} // }}}

func (d *Dao) getCacheKey(opts []db.FnSqlOption) []byte { // {{{
	so := &db.SqlOption{}
	for _, opt := range opts {
		opt(so)
	}

	var sb strings.Builder

	if sql := so.GetSql(); sql != "" {
		sb.WriteString(sql)
		sb.WriteString("#")
	} else {
		sb.WriteString(so.GetFields())
		sb.WriteString("#")
		sb.WriteString(so.GetTable())
		sb.WriteString("#")
		sb.WriteString(so.GetAlias())
		sb.WriteString("#")
		sb.WriteString(so.GetIdx())
		sb.WriteString("#")
		sb.WriteString(strings.Join(so.GetLeftJoin(), ","))
		sb.WriteString("#")
		sb.WriteString(strings.Join(so.GetInnerJoin(), ","))
		sb.WriteString("#")
		sb.WriteString(so.GetWhere())
		sb.WriteString("#")
		sb.WriteString(so.GetGroup())
		sb.WriteString("#")
		sb.WriteString(so.GetOrder())
		sb.WriteString("#")
		sb.WriteString(so.GetLimits())
	}

	for _, v := range so.GetVals() {
		sb.WriteString(",")
		sb.WriteString(x.AsString(v))
	}

	if so.GetUseBytes() {
		sb.WriteString("#1")
	}

	return []byte(sb.String())
} // }}}

type CacheData struct {
	Num       int
	Res       any
	CacheTime time.Time
}
