package tx

import (
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/db"
)

// opts: confName, [isReadOnly], 最后一个参数如果为bool值，则表示是否开启只读事务
func TransBegin(opts ...any) (db.DBClient, error) {
	conf_name := "db_master"
	is_readonly := false

	l := len(opts)
	if l > 0 {
		if v, ok := opts[0].(string); ok {
			conf_name = v
		}

		if v, ok := opts[l-1].(bool); ok {
			is_readonly = v
		}
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
}
