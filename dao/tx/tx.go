package tx

import (
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/db"
)

// 开启事务，参数: DB配置名
func TransBegin(conf_names ...string) (db.DBClient, error) {
	return transBegin(false, conf_names...)
}

// 开启只读事务
func ReadTransBegin(conf_names ...string) (db.DBClient, error) {
	return transBegin(true, conf_names...)
}

func transBegin(is_readonly bool, conf_names ...string) (db.DBClient, error) { // {{{
	conf_name := "db_master"

	if len(conf_names) > 0 {
		conf_name = conf_names[0]
	}

	conf := x.Conf.GetMap(conf_name)
	if 0 == len(conf) {
		panic("db资源不存在: " + conf_name)
	}

	tx, err := x.DB.Get(conf)
	if err != nil {
		return nil, err
	}

	return tx.Begin(is_readonly)
} // }}}
