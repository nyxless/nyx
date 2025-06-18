package x

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nyxless/nyx/x/db"
	"strings"
	"sync"
	// 新增 driver 在项目中使用 RegisterSqlDriver 注册
	//	_ "github.com/mattn/go-sqlite3"
	//	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func NewDBProxy() *DBProxy {
	return &DBProxy{c: map[string]*db.SqlClient{}}
}

type DBProxy struct {
	mutex sync.RWMutex
	c     map[string]*db.SqlClient
}

type SqlDsn func(MAP) string

// type NewDBFunc func() db.DBClient
type NewDBFunc func() *db.SqlClient

func RegisterSqlDriver(name string, dsnfunc SqlDsn, f ...NewDBFunc) {
	dsnFuncs[name] = dsnfunc
	if len(f) > 0 {
		newDBFunc = f[0]
	}
}

var newDBFunc NewDBFunc = db.NewSqlClient

var dsnFuncs = map[string]SqlDsn{
	"mysql": func(conf MAP) string {

		host := AsString(conf["host"])
		user := AsString(conf["user"])
		password := AsString(conf["password"])
		database := AsString(conf["database"])
		charset := AsString(conf["charset"])

		return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s", user, password, host, database, charset)
	},
}

func (this *DBProxy) Get(conf MAP) *db.SqlClient { // {{{
	this.mutex.RLock()
	v, ok := this.c[AsString(conf["host"])]
	this.mutex.RUnlock()

	if ok && v.Ping() != nil {
		v.Close()
		ok = false
	}

	if !ok {
		v = this.add(conf)
	}

	return v
} // }}}

func (this *DBProxy) add(conf MAP) *db.SqlClient { // {{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var sqlClient *db.SqlClient
	var err error
	var debug bool
	var dsn string

	if Debug { //全局 debug 开关, 启动时 -d
		debug = true
	} else {
		debug = AsBool(conf["debug"])
	}

	max_open_conns := AsInt(conf["max_open_conns"])
	max_idle_conns := AsInt(conf["max_idle_conns"])

	dbt := strings.ToLower(AsString(conf["type"]))

	if dsnfunc, ok := dsnFuncs[dbt]; ok {
		dsn = dsnfunc(conf)
	} else {
		panic("不支持的db类型:" + dbt)
	}

	var _db *sql.DB
	_db, err = sql.Open(dbt, dsn)
	if err != nil {
		panic(fmt.Sprintf("%s connect error: %v", dbt, err))
	}

	if max_open_conns > 0 {
		_db.SetMaxOpenConns(max_open_conns)
	}

	if max_idle_conns > 0 {
		_db.SetMaxIdleConns(max_idle_conns)
	}

	sqlClient = newDBFunc()
	sqlClient.SetDB(dbt, _db)
	sqlClient.SetDebug(debug)

	this.c[AsString(conf["host"])] = sqlClient

	Printf("add dbproxy: host [ %s ], type [ %s ] db [ %s ], #ID [ %s ]\n", conf["host"], dbt, conf["database"], sqlClient.ID())

	return sqlClient
} // }}}

// 拼装参数时，作为可执行字符，而不是字符串值
func Expr(param string) string { // {{{
	if "" != param {
		return "#~#" + param
	}

	return ""
} // }}}
