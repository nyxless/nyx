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

func (this *Dao) Init(conf_name ...string) { //{{{
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

	this.defaultFields = "*"
	this.filterValues = []any{}

	this.DBWriter, err = x.DB.Get(master_conf)
	if err != nil {
		x.Panic(err)
	}

	this.DBReader, err = x.DB.Get(slave_conf)
	if err != nil {
		x.Panic(err)
	}

	if this.DBWriter.Type() == "mysql" {
		this.autoOrder = true
	}

	gob.Register(&CacheData{})
	gob.Register(time.Time{})
	gob.Register(map[string]interface{}{})
	gob.Register([]map[string]interface{}{})
} // }}}

func (this *Dao) InitTx(tx db.DBClient) { //使用事务{{{
	this.defaultFields = "*"
	this.filterValues = []any{}
	this.autoOrder = true
	this.DBWriter = tx
	this.DBReader = tx
} // }}}

func (this *Dao) WithContext(ctx context.Context) *Dao {
	this.ctx = ctx

	return this
}

func (this *Dao) SetTable(table string) {
	this.table = table
}

func (this *Dao) GetTable() string {
	return this.table
}

func (this *Dao) SetPrimary(field string) {
	this.primary = field
}

func (this *Dao) GetPrimary() string {
	return this.primary
}

func (this *Dao) SetCountField(field string) *Dao { // {{{
	this.countField = field
	return this
} // }}}

func (this *Dao) GetCountField() string { // {{{
	field := "1"
	if "" != this.countField {
		field = this.countField
		this.countField = ""
	}

	return field
} // }}}

func (this *Dao) SetDefaultFields(fields ...string) *Dao { // {{{
	this.defaultFields = strings.Join(fields, ",")
	return this
} // }}}

// 可在读方法前使用，且仅对本次查询起作用，如: NewDAOUser().SetFields("uid").GetRecord(uid)
func (this *Dao) SetFields(fields ...string) *Dao {
	this.fields = strings.Join(fields, ",")
	return this
}

func (this *Dao) GetFields() string { // {{{
	fields := this.defaultFields
	if "" != this.fields {
		fields = this.fields
		this.fields = ""
	}

	return fields
} // }}}

func (this *Dao) UseIndex(idx string) *Dao {
	this.index = idx
	return this
}

// 支持 GetRecords 方法, 返回符合条件的总数, 赋值给参数 cnt
func (this *Dao) WithCount(cnt *int) *Dao {
	this.cnt = cnt

	return this
}

func (this *Dao) getIndex() string { // {{{
	idx := this.index
	this.index = ""
	return idx
} // }}}

// 强制使用主库
func (this *Dao) UseMaster(flag ...bool) *Dao { // {{{
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	this.forceMaster = use
	return this
} // }}}

// 返回缓存时间
func (this *Dao) GetCacheTime() (time.Time, bool) { // {{{
	cacheTime := this.cacheTime
	hit := this.hit
	this.cacheTime = time.Time{}
	this.hit = false

	return cacheTime, hit
} // }}}

func (this *Dao) WithCache(ttl int, callbackFns ...CacheCallbackFn) *Dao { // {{{
	this.useCache = true
	this.cacheTtl = ttl

	if len(callbackFns) > 0 {
		this.cacheCallbackFn = callbackFns[0]
	}

	return this
} // }}}

func (this *Dao) WithRefreshCache(refreshInterval int, callbackFns ...CacheCallbackFn) *Dao { // {{{
	this.useCache = true
	this.cacheRefreshInterval = refreshInterval

	if len(callbackFns) > 0 {
		this.cacheCallbackFn = callbackFns[0]
	}

	return this
} // }}}

func (this *Dao) getUseCache() (bool, int, int, CacheCallbackFn) { // {{{
	use_cache := this.useCache
	ttl := this.cacheTtl
	refreshInterval := this.cacheRefreshInterval
	callbackFn := this.cacheCallbackFn

	this.useCache = false
	this.cacheTtl = 0
	this.cacheRefreshInterval = 0
	this.cacheCallbackFn = nil

	return use_cache, ttl, refreshInterval, callbackFn
} // }}}

func (this *Dao) SetAutoOrder(flag ...bool) *Dao { // {{{
	use := true
	if len(flag) > 0 {
		use = flag[0]
	}

	this.autoOrder = use
	return this
} // }}}

func (this *Dao) Order(order ...string) *Dao { // {{{
	this.order = strings.Join(order, ",")
	this.autoOrder = false
	return this
} // }}}

func (this *Dao) getOrder(use_auto_order bool) string { // {{{
	order := this.order
	if "" == order && use_auto_order && this.autoOrder {
		order = this.GetPrimary() + " desc"
	}

	this.order = ""
	this.autoOrder = true

	return order
} // }}}

func (this *Dao) Group(group ...string) *Dao { // {{{
	this.group = strings.Join(group, ",")
	return this
} // }}}

func (this *Dao) getGroup() string { // {{{
	group := this.group
	this.group = ""

	return group
} // }}}

func (this *Dao) Limit(limit int, limits ...int) *Dao { // {{{
	this.limit = strconv.Itoa(limit)

	if len(limits) > 0 {
		this.limit = this.limit + "," + strconv.Itoa(limits[0])
	}

	return this
} // }}}

func (this *Dao) getLimit() string { // {{{
	limit := this.limit
	this.limit = ""

	return limit
} // }}}

func (this *Dao) Alias(alias string) *Dao { // {{{
	this.alias = alias
	return this
} // }}}

func (this *Dao) getAlias() string { // {{{
	alias := this.alias
	this.alias = ""

	return alias
} // }}}

func (this *Dao) LeftJoin(left_join ...*JoinOn) *Dao { // {{{
	this.leftJoin = append(this.leftJoin, left_join...)

	return this
} // }}}

func (this *Dao) InnerJoin(inner_join ...*JoinOn) *Dao { // {{{
	this.innerJoin = append(this.innerJoin, inner_join...)

	return this
} // }}}

func (this *Dao) On(left_field string, right_fields ...string) *JoinOn { // {{{
	return this.on("=", left_field, right_fields...)
} // }}}

func (this *Dao) NotOn(left_field string, right_fields ...string) *JoinOn { // {{{
	return this.on("!=", left_field, right_fields...)
} // }}}

// compare: 比较符号
func (this *Dao) CompareOn(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	return this.on(compare, left_field, right_fields...)
} // }}}

func (this *Dao) on(compare, left_field string, right_fields ...string) *JoinOn { // {{{
	right_field := left_field
	if len(right_fields) > 0 {
		right_field = right_fields[0]
	}

	return &JoinOn{this, []*OnPair{&OnPair{left_field, right_field, compare}}}
} // }}}

func (this *Dao) GetDBReader() db.DBClient { // {{{
	if this.forceMaster {
		this.forceMaster = false

		return this.DBWriter
	}

	return this.DBReader
} // }}}

// 解析where条件
// 例1:parseParams("x=? and y=?", 1, 2)
// 例2:parseParams("x=? and y=?", []any{1,2}) 等价于 parseParams("a=? and b=?", 1, 2) //若第二个参数非[]any(如[]int、[]string), 可先使用 AsSlice 进行转换
// 例3:parseParams(map[string]any{"a":1,"b":2}) 等价于 parseParams("a=? and b=?", 1, 2)
// 例4:parseParams(map[string]any{"a":1,"b":[]any{2, 3}}) 等价于 parseParams("a=? and b in ('2','3')", 1)
func (this *Dao) parseParams(params ...any) (string, []any) { //{{{
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

func (this *Dao) SetFilter(params ...any) *Dao { //{{{
	where, values := this.parseParams(params...)

	if where != "" {
		if this.filter != "" {
			this.filter += " and " + where
		} else {
			this.filter = where
		}

		if len(values) > 0 {
			this.filterValues = append(this.filterValues, values...)
		}
	}

	return this
} //}}}

func (this *Dao) getFilter() (string, []any) { // {{{
	where := this.filter
	this.filter = ""

	values := this.filterValues
	this.filterValues = []any{}

	return where, values
} // }}}

// 在主库执行 sql 操作
func (this *Dao) Execute(sql string, params ...any) (int, error) { //{{{
	return this.DBWriter.Execute(sql, params...)
} // }}}

// 在从库执行 sql 查询单字段, 返回 any
func (this *Dao) QueryOne(sql string, params ...any) (any, error) { //{{{
	return this.GetDBReader().QueryOne(sql, params...)
} // }}}

// 在从库执行 sql 查询单行, 返回 Map
func (this *Dao) QueryRow(sql string, params ...any) (map[string]any, error) { //{{{
	return this.GetDBReader().QueryRow(sql, params...)
} // }}}

// 在从库执行 sql 查询, 返回 MapSlice
func (this *Dao) Query(sql string, params ...any) ([]map[string]any, error) { //{{{
	return this.GetDBReader().Query(sql, params...)
} // }}}

// 返回迭代器
func (this *Dao) QueryStream(sql string, params ...any) (*db.RowIter, error) { //{{{
	return this.GetDBReader().QueryStream(sql, params...)
} // }}}

// 插入新记录
func (this *Dao) AddRecord(vals ...map[string]any) (int, error) { //{{{
	return this.DBWriter.Insert(this.table, vals...)
} // }}}

// 更新记录
func (this *Dao) SetRecord(vals map[string]any, id any) (int, error) { //{{{
	delete(vals, this.primary)
	return this.DBWriter.Update(this.table, vals, this.primary+"=?", id)
} // }}}

// 按条件更新
func (this *Dao) SetRecordBy(vals map[string]any, where string, params ...any) (int, error) { //{{{
	return this.DBWriter.Update(this.table, vals, where, params...)
} // }}}

// upsert 操作
func (this *Dao) ResetRecord(vals map[string]any) (int, error) { //{{{
	return this.DBWriter.Upsert(this.table, vals, this.primary)
} // }}}

// 按主键查询记录
func (this *Dao) GetRecord(id any) (map[string]any, error) { //{{{
	alias := this.getAlias()
	primary := this.primary
	join_table := this.table

	if alias != "" {
		primary = alias + "." + primary
		join_table = alias
	}

	left_join := parseJoinOn(join_table, this.leftJoin)
	inner_join := parseJoinOn(join_table, this.innerJoin)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(this.GetFields()),
		db.WithAlias(alias),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
		db.WithWhere(primary+"=?", []any{id}),
	}

	res, err := this.getCache(func() (int, any, error) {

		row, err := this.GetDBReader().GetRow(sqlOptions...)
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
func (this *Dao) DelRecord(id any) (int, error) { //{{{
	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithWhere(this.primary+"=?", []any{id}),
		db.WithLimits("1"),
	}

	return this.DBWriter.Delete(sqlOptions...)
} // }}}

// 删除符合条件的数据 (一条)
func (this *Dao) DelRecordBy(params ...any) (int, error) { //{{{
	this.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithOrder(this.getOrder(false)),
		db.WithWhere(this.getFilter()),
		db.WithLimits("1"),
	}

	return this.DBWriter.Delete(sqlOptions...)
} // }}}

// 删除所有符合条件的数据 (Is Dangerous!)
func (this *Dao) DelRecords(params ...any) (int, error) { //{{{
	this.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithWhere(this.getFilter()),
	}

	return this.DBWriter.Delete(sqlOptions...)
} // }}}

func (this *Dao) GetOne(field string, params ...any) (any, error) { //{{{
	this.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(field),
		db.WithIdx(this.getIndex()),
		db.WithGroup(this.getGroup()),
		db.WithOrder(this.getOrder(false)),
		db.WithWhere(this.getFilter()),
	}

	res, err := this.getCache(func() (int, any, error) {
		res, err := this.GetDBReader().GetOne(sqlOptions...)
		return 0, res, err
	}, sqlOptions)

	return res, err
} // }}}

// alias for GetOne
func (this *Dao) GetValue(field string, params ...any) (any, error) { //{{{
	return this.GetOne(field, params...)
} // }}}

func (this *Dao) GetValues(field string, params ...any) ([]any, error) { //{{{
	this.SetFilter(params...)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(field),
		db.WithIdx(this.getIndex()),
		db.WithGroup(this.getGroup()),
		db.WithOrder(this.getOrder(false)),
		db.WithWhere(this.getFilter()),
	}

	res, err := this.getCache(func() (int, any, error) {

		list, err := this.GetDBReader().GetAll(sqlOptions...)
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

func (this *Dao) GetValuesMap(keyfield, valfield string, params ...any) (map[any]any, error) { //{{{
	this.SetFilter(params...)

	field := this.GetFields()
	if field == this.defaultFields {
		field = keyfield + "," + valfield
	}

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(field),
		db.WithIdx(this.getIndex()),
		db.WithGroup(this.getGroup()),
		db.WithOrder(this.getOrder(false)),
		db.WithWhere(this.getFilter()),
	}

	res, err := this.getCache(func() (int, any, error) {

		list, err := this.GetDBReader().GetAll(sqlOptions...)
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

func (this *Dao) GetCount(params ...any) (int, error) { //{{{
	this.SetFilter(params...)

	field := "count(" + this.GetCountField() + ") as total"

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(field),
		db.WithIdx(this.getIndex()),
		db.WithGroup(this.getGroup()),
		db.WithWhere(this.getFilter()),
	}

	res, err := this.getCache(func() (int, any, error) {

		count, err := this.GetDBReader().GetOne(sqlOptions...)
		if err != nil {
			return 0, nil, err
		}
		return 0, count, nil

	}, sqlOptions)

	return x.AsInt(res), err
} // }}}

func (this *Dao) Exists(id any) (bool, error) { //{{{
	one, err := this.GetOne(this.primary, this.primary+"=?", id)
	if err != nil {
		return false, err
	}

	return one != nil, nil
} // }}}

func (this *Dao) ExistsBy(params ...any) (bool, error) { //{{{
	one, err := this.GetOne(this.primary, params...)
	if err != nil {
		return false, err
	}

	return one != nil, nil
} // }}}

func (this *Dao) GetRecordBy(params ...any) (map[string]any, error) { //{{{
	this.SetFilter(params...)

	alias := this.getAlias()
	join_table := this.table

	if alias != "" {
		join_table = alias
	}

	left_join := parseJoinOn(join_table, this.leftJoin)
	inner_join := parseJoinOn(join_table, this.innerJoin)

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(this.GetFields()),
		db.WithAlias(alias),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
		db.WithWhere(this.getFilter()),
	}

	res, err := this.getCache(func() (int, any, error) {

		row, err := this.GetDBReader().GetRow(sqlOptions...)
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

func (this *Dao) GetRecords(params ...any) ([]map[string]any, error) { //{{{
	this.SetFilter(params...)

	alias := this.getAlias()
	join_table := this.table

	if alias != "" {
		join_table = alias
	}

	left_join := parseJoinOn(join_table, this.leftJoin)
	inner_join := parseJoinOn(join_table, this.innerJoin)

	field := this.GetFields()

	idx := this.getIndex()
	group := this.getGroup()
	where, values := this.getFilter()

	sqlOptions := []db.FnSqlOption{
		db.WithTable(this.table),
		db.WithFields(field),
		db.WithAlias(alias),
		db.WithIdx(idx),
		db.WithGroup(group),
		db.WithOrder(this.getOrder(true)),
		db.WithLimits(this.getLimit()),
		db.WithLeftJoin(left_join),
		db.WithInnerJoin(inner_join),
		db.WithWhere(where, values),
	}

	getRecordsFn := func() (int, any, error) {
		res, err := this.GetDBReader().GetAll(sqlOptions...)
		return 0, res, err
	}

	// 查询列表 + 总数, 总数赋值到指定变量cnt
	// *mysql 支持 FOUND_ROWS(), 但大数据下可能存在性能问题，因此使用更兼容的方案
	if this.cnt != nil {
		getRecordsFn = func() (int, any, error) {
			var count any
			var list []map[string]any
			var err error

			db_reader := this.GetDBReader()
			list, err = db_reader.GetAll(sqlOptions...)

			if err != nil {
				return 0, nil, err
			}

			count, err = db_reader.GetOne(db.WithTable(this.table), db.WithFields("count(1) as total"), db.WithIdx(idx), db.WithGroup(group), db.WithWhere(where, values))
			if err != nil {
				return 0, nil, err
			}

			return x.AsInt(count), list, nil
		}
	}

	res, err := this.getCache(getRecordsFn, sqlOptions)

	if err != nil {
		return nil, err
	}

	return res.([]map[string]any), err
} // }}}

func (this *Dao) getCache(fn func() (int, any, error), opts []db.FnSqlOption) (res any, err error) { //{{{
	use_cache, ttl, refreshInterval, callbackFn := this.getUseCache()

	var num int
	if use_cache && x.LocalCache != nil {
		cache_data, hit, err := this.getFromCache(fn, opts, ttl, refreshInterval, callbackFn)
		if err != nil {
			return nil, err
		}
		num = cache_data.Num
		res = cache_data.Res

		this.hit = hit
		this.cacheTime = cache_data.CacheTime
	} else {
		num, res, err = fn()
	}

	if this.cnt != nil {
		*this.cnt = num
		this.cnt = nil
	}

	return res, err
} // }}}

func (this *Dao) getFromCache(fn func() (int, any, error), opts []db.FnSqlOption, ttl, refreshInterval int, callbackFn CacheCallbackFn) (*CacheData, bool, error) { // {{{
	key := this.getCacheKey(opts)
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

			return callbackFn(cache_data)
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

func (this *Dao) getCacheKey(opts []db.FnSqlOption) []byte { // {{{
	so := &db.SqlOption{}
	for _, opt := range opts {
		opt(so)
	}

	var sb strings.Builder

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

	for _, v := range so.GetVals() {
		sb.WriteString(",")
		sb.WriteString(x.AsString(v))
	}

	return []byte(sb.String())
} // }}}

type CacheData struct {
	Num       int
	Res       any
	CacheTime time.Time
}
