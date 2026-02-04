package x

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nyxless/nyx/x/db"
	"golang.org/x/sync/singleflight"
	"strings"
	"sync"
	"time"
	// 新增 driver 在项目中使用 RegisterSqlDriver 注册
	//	_ "github.com/mattn/go-sqlite3"
	//	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func NewDBProxy() *DBProxy {
	return &DBProxy{
		c:  make(map[string]db.DBClient),
		sf: &singleflight.Group{},
	}
}

type DBProxy struct {
	mutex sync.RWMutex
	c     map[string]db.DBClient
	sf    *singleflight.Group
}

type SqlDsn func(MAP) string

type NewDBFunc func() db.DBClient

func RegisterSqlDriver(name string, dsnfunc SqlDsn, f ...NewDBFunc) {
	dsnFuncs[name] = dsnfunc
	if len(f) > 0 {
		newDBFunc = f[0]
	}
}

var newDBFunc NewDBFunc = db.NewDBClient

var dsnFuncs = map[string]SqlDsn{
	"mysql": func(conf MAP) string {

		host := AsString(conf["host"])
		user := AsString(conf["user"])
		password := AsString(conf["password"])
		database := AsString(conf["database"])
		charset := AsString(conf["charset"])

		return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=true&loc=%s", user, password, host, database, charset, TIME_ZONE)
	},
}

func (this *DBProxy) Get(conf MAP) (db.DBClient, error) { // {{{
	host, ok := conf["host"]
	if !ok {
		return nil, fmt.Errorf("DB 配置有误")
	}

	key := AsString(host)
	if client := this.getClient(key); client != nil {
		return client, nil
	}

	result, err, _ := this.sf.Do(key, func() (interface{}, error) {
		// 再次检查，防止在等待期间已经有其他goroutine创建了连接
		if client := this.getClient(key); client != nil {
			return client, nil
		}

		// 创建新连接
		return this.add(conf, key)
	})

	if err != nil {
		return nil, err
	}

	return result.(db.DBClient), nil

} // }}}

func (this *DBProxy) getClient(key string) db.DBClient { // {{{
	this.mutex.RLock()
	client, ok := this.c[key]
	this.mutex.RUnlock()

	if !ok || client == nil {
		return nil
	}

	return client
} // }}}

func (this *DBProxy) add(conf MAP, key string) (db.DBClient, error) { // {{{
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
	conn_max_idle_time := AsInt(conf["conn_max_idle_time"])
	conn_max_lifetime := AsInt(conf["conn_max_lifetime"])

	dbt := strings.ToLower(AsString(conf["type"]))

	if dsnfunc, ok := dsnFuncs[dbt]; ok {
		dsn = dsnfunc(conf)
	} else {
		return nil, fmt.Errorf("不支持的db类型: %s", dbt)
	}

	var _db *sql.DB
	_db, err = sql.Open(dbt, dsn)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %v", dbt, err)
	}

	if max_open_conns > 0 {
		_db.SetMaxOpenConns(max_open_conns)
	}

	if max_idle_conns > 0 {
		_db.SetMaxIdleConns(max_idle_conns)
	}

	if conn_max_idle_time > 0 {
		_db.SetConnMaxIdleTime(time.Duration(conn_max_idle_time) * time.Second)
	}

	if conn_max_lifetime > 0 {
		_db.SetConnMaxLifetime(time.Duration(conn_max_lifetime) * time.Second)
	}

	client := newDBFunc()
	client.SetDB(dbt, _db)
	client.SetDebug(debug)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	err = client.Ping(ctx)
	cancel()

	if err != nil {
		client.Close()
		return nil, fmt.Errorf("无法连接到 DB: [%v] %v", conf["host"], err)
	}

	this.mutex.Lock()
	this.c[key] = client
	this.mutex.Unlock()

	Info("Add DBProxy:", fmt.Sprintf(" host [ %s ], type [ %s ] db [ %s ], #ID [ %s ]", conf["host"], dbt, conf["database"], client.ID()))

	return client, nil
} // }}}

func (this *DBProxy) Close() { // {{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, client := range this.c {
		client.Close()
	}
	this.c = make(map[string]db.DBClient)
} // }}}

// 拼装参数时，作为可执行字符，而不是字符串值
func Expr(param string) string { // {{{
	return db.Expr(param)
} // }}}
