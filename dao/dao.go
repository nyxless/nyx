package dao

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/db"
	"strconv"
	"strings"
	"time"
)

type Dao struct {
	DBWriter, DBReader   db.DBClient
	table                string
	primary              string
	defaultFields        string //默认字段,逗号分隔
	fields               string //通过setFields方法指定的字段,逗号分隔,只能通过getFields使用一次
	countField           string //getCount方法使用的字段
	index                string //查询使用的索引
	limit                string
	autoOrder            bool //是否自动排序(默认按自动主键倒序排序)
	order                string
	group                string
	filter               string //过滤条件
	filterValues         []any  //过滤条件的值
	forceMaster          bool   //强制使用主库读，只能通过useMaster 使用一次
	ctx                  context.Context
	alias                string //表别名
	leftJoin             []*JoinOn
	innerJoin            []*JoinOn
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

type JoinOn struct {
	JoinDao *Dao
	OnPairs []*OnPair
}
type OnPair struct {
	Left    string
	Right   string
	Compare string
}

func (j *JoinOn) On(left_field string, right_fields ...string) *JoinOn {
	return j.on("=", left_field, right_fields...)
}

func (j *JoinOn) NotOn(left_field string, right_fields ...string) *JoinOn {
	return j.on("!=", left_field, right_fields...)
}

// compare: 比较符号
func (j *JoinOn) CompareOn(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	return j.on(compare, left_field, right_fields...)
} // }}}

func (j *JoinOn) on(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	right_field := left_field
	if len(right_fields) > 0 {
		right_field = right_fields[0]
	}

	j.OnPairs = append(j.OnPairs, &OnPair{left_field, right_field, compare})

	return j
} // }}}

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
	d.filterValues = []any{}

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
	d.filterValues = []any{}
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
	field := "1"
	if "" != d.countField {
		field = d.countField
		d.countField = ""
	}

	return field
} // }}}

func (d *Dao) SetDefaultFields(fields ...string) *Dao { // {{{
	d.defaultFields = strings.Join(fields, ",")
	return d
} // }}}

// 可在读方法前使用，且仅对本次查询起作用，如: NewDAOUser().SetFields("uid").GetRecord(uid)
func (d *Dao) SetFields(fields ...string) *Dao {
	d.fields = strings.Join(fields, ",")
	return d
}

func (d *Dao) GetFields() string { // {{{
	fields := d.defaultFields
	if "" != d.fields {
		fields = d.fields
		d.fields = ""
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
	d.order = strings.Join(order, ",")
	d.autoOrder = false
	return d
} // }}}

func (d *Dao) getOrder(alias string, use_auto_order bool) string { // {{{
	order := d.order
	if "" == order && use_auto_order && d.autoOrder {
		order = d.GetPrimary() + " desc"
		if alias != "" {
			order = alias + "." + order
		}
	}

	d.order = ""
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

func (d *Dao) getAlias() string { // {{{
	alias := d.alias
	d.alias = ""

	return alias
} // }}}

func (d *Dao) LeftJoin(left_join ...*JoinOn) *Dao { // {{{
	d.leftJoin = append(d.leftJoin, left_join...)

	return d
} // }}}

func (d *Dao) InnerJoin(inner_join ...*JoinOn) *Dao { // {{{
	d.innerJoin = append(d.innerJoin, inner_join...)

	return d
} // }}}

func (d *Dao) On(left_field string, right_fields ...string) *JoinOn { // {{{
	return d.on("=", left_field, right_fields...)
} // }}}

func (d *Dao) NotOn(left_field string, right_fields ...string) *JoinOn { // {{{
	return d.on("!=", left_field, right_fields...)
} // }}}

// compare: 比较符号
func (d *Dao) CompareOn(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	return d.on(compare, left_field, right_fields...)
} // }}}

func (d *Dao) on(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	right_field := left_field
	if len(right_fields) > 0 {
		right_field = right_fields[0]
	}

	return &JoinOn{d, []*OnPair{&OnPair{left_field, right_field, compare}}}
} // }}}

func (d *Dao) GetDBReader() db.DBClient { // {{{
	if d.forceMaster {
		d.forceMaster = false

		return d.DBWriter
	}

	return d.DBReader
} // }}}

// 解析where条件
// 例1:parseParams("x=? and y=?", 1, 2)
// 例2:parseParams("x=? and y=?", []any{1,2}) 等价于 parseParams("a=? and b=?", 1, 2) //若第二个参数非[]any(如[]int、[]string), 可先使用 AsSlice 进行转换
// 例3:parseParams(map[string]any{"a":1,"b":2}) 等价于 parseParams("a=? and b=?", 1, 2)
// 例4:parseParams(map[string]any{"a":1,"b":[]any{2, 3}}) 等价于 parseParams("a=? and b in ('2','3')", 1)
func (d *Dao) parseParams(params ...any) (string, []any) { //{{{
	where := ""
	values := []any{}

	l := len(params)
	if l > 0 {
		switch val := params[0].(type) {
		case string:
			where = val
			values = params[1:]
			if l == 2 {
				if slice_val, ok := params[1].([]any); ok {
					values = slice_val
				}
			}
		case map[string]any:
			for k, v := range val {
				if where != "" {
					where = where + " and "
				}

				if slice_val, ok := v.([]any); ok {
					where = where + "`" + k + "` in ("

					for i, j := range slice_val {
						if i > 0 {
							where = where + ","
						}
						where = where + "'" + strings.ReplaceAll(x.AsString(j), `'`, `\'`) + "'"
					}
					where = where + ")"
				} else {
					where = where + "`" + k + "`=?"
					values = append(values, v)
				}
			}
		default:
			where = x.AsString(params[0])
			values = params[1:]
		}
	}

	return where, values
} //}}}

func (d *Dao) SetFilter(params ...any) *Dao { //{{{
	where, values := d.parseParams(params...)

	if where != "" {
		if d.filter != "" {
			d.filter += " and " + where
		} else {
			d.filter = where
		}

		if len(values) > 0 {
			d.filterValues = append(d.filterValues, values...)
		}
	}

	return d
} //}}}

func (d *Dao) getFilter() (string, []any) { // {{{
	where := d.filter
	d.filter = ""

	values := d.filterValues
	d.filterValues = []any{}

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

// 插入新记录
func (d *Dao) AddRecord(vals ...map[string]any) (int, error) { //{{{
	return d.DBWriter.Insert(d.table, vals...)
} // }}}

// 更新记录
func (d *Dao) SetRecord(vals map[string]any, id any) (int, error) { //{{{
	delete(vals, d.primary)
	return d.DBWriter.Update(d.table, vals, d.primary+"=?", id)
} // }}}

// 按条件更新
func (d *Dao) SetRecordBy(vals map[string]any, where string, params ...any) (int, error) { //{{{
	return d.DBWriter.Update(d.table, vals, where, params...)
} // }}}

// upsert 操作
func (d *Dao) ResetRecord(vals map[string]any) (int, error) { //{{{
	return d.DBWriter.Upsert(d.table, vals, d.primary)
} // }}}

// 按主键查询记录
func (d *Dao) GetRecord(id any) (map[string]any, error) { //{{{
	alias := d.getAlias()
	primary := d.primary
	join_table := d.table

	if alias != "" {
		primary = alias + "." + primary
		join_table = alias
	}

	left_join := parseJoinOn(join_table, d.leftJoin)
	inner_join := parseJoinOn(join_table, d.innerJoin)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(alias),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
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

func parseJoinOn(alias string, joinons []*JoinOn) []string { // {{{
	var joins []string
	for _, v := range joinons {
		var join string

		tbl := v.JoinDao.GetTable()
		al := v.JoinDao.getAlias()
		join += tbl + " " + al
		if al == "" {
			al = tbl
		}
		for i, p := range v.OnPairs {
			if i == 0 {
				join += " ON "
			} else {
				join += " AND "
			}

			on := "="
			if p.Compare != "" {
				on = p.Compare
			}

			join += alias + "." + p.Left + on + al + "." + p.Right
		}

		joins = append(joins, join)
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
		db.WithOrder(d.getOrder("", false)),
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
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder("", false)),
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
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder("", false)),
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

func (d *Dao) GetValuesMap(keyfield, valfield string, params ...any) (map[any]any, error) { //{{{
	d.SetFilter(params...)

	field := d.GetFields()
	if field == d.defaultFields {
		field = keyfield + "," + valfield
	}

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithIdx(d.getIndex()),
		db.WithGroup(d.getGroup()),
		db.WithOrder(d.getOrder("", false)),
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

func (d *Dao) GetCount(params ...any) (int, error) { //{{{
	d.SetFilter(params...)

	field := "count(" + d.GetCountField() + ") as total"

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
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

	alias := d.getAlias()
	join_table := d.table

	if alias != "" {
		join_table = alias
	}

	left_join := parseJoinOn(join_table, d.leftJoin)
	inner_join := parseJoinOn(join_table, d.innerJoin)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(d.GetFields()),
		db.WithAlias(alias),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
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

	alias := d.getAlias()
	join_table := d.table

	if alias != "" {
		join_table = alias
	}

	left_join := parseJoinOn(join_table, d.leftJoin)
	inner_join := parseJoinOn(join_table, d.innerJoin)

	field := d.GetFields()

	idx := d.getIndex()
	group := d.getGroup()
	where, values := d.getFilter()

	sqlOptions := []db.FnSqlOption{
		db.WithTable(d.table),
		db.WithFields(field),
		db.WithAlias(alias),
		db.WithIdx(idx),
		db.WithGroup(group),
		db.WithOrder(d.getOrder(alias, true)),
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

			count, err = db_reader.GetOne(db.WithTable(d.table), db.WithFields("count(1) as total"), db.WithIdx(idx), db.WithGroup(group), db.WithWhere(where, values))
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
