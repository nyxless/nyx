package x

import (
	"github.com/nyxless/nyx/x/cache"
	"github.com/nyxless/nyx/x/db"
	"github.com/nyxless/nyx/x/log"
)

// 应用程序根路径
var AppRoot string

// 全局 Debug 开关
var Debug bool

// 在项目代码中指定时区
var TIME_ZONE = "Local" // Asia/Shanghai, UTC

// 全局共用配置
var Conf *Config

// 全局共用日志
var Logger *log.Logger

// 本地缓存
var LocalCache *cache.Cache

// 全局共用db代理
var DB = NewDBProxy()

// 全局共用redis代理
var Redis = NewRedisProxy()

// 配置文件变量缓存(框架中使用)，初始化时赋值
var (
	Conf_env_mode                      string
	Conf_template_enabled              bool
	Conf_max_post_size                 int64
	Conf_access_log_enabled            bool
	Conf_access_log_success_level_name string
	Conf_access_log_error_level_name   string
	Conf_access_log_omit_params        []string
	Conf_rpc_auth                      map[string]string
	Conf_default_controller            string
	Conf_default_action                string

	//由路由配置衍生的map
	/*
		//不带参数的路由配置
		UrlRoutes = map[string]map[string]string{
			"/api/v1/user/getUserInfo" : map[string]string{
				"method":  map[string]string{"GET":"GET", "POST":"POST"},
				"controller": "user",
				"action":   "getUserInfo",
			},
		}

		//带参数的路由配置: 如: /api/v1/users/user/getUserInfo/@id
		UrlParamRoutes = map[int]map[int]map[string]map[string]any{
			6 => map[int]map[string]map[string]any{ //path num: 6
				1 => map[string]map[string]any{ //param num:  1
					"/api/v1/users/user/getUserInfo" : map[string]any{
						"method":  map[string]string{"GET":"GET", "POST":"POST"},
						"controller": "user",
						"action":   "getUserInfo",
						"params": []string{"id"}, //多个参数时，倒序
					},
				},
			},
		}

		//前缀替换
		UrlPrefix = []map[string]string{map[string]string{"from":"api/v1", "to":""}}
	*/

	UrlRoutes      map[string]map[string]any
	UrlParamRoutes map[int]map[int]map[string]map[string]any
	UrlPrefix      []map[string]string
)

// 方便直接从x引用
type SqlClient = db.SqlClient

// 简化业务代码
// 使用MAP替代map[string]any
type MAP = map[string]any

// 使用MAPS替代map[string]string
type MAPS = map[string]string

// 使用MAPI替代map[string]int
type MAPI = map[string]int

// 使用MAP替代map[int]any
type IMAP = map[int]any

// 使用IMAPS替代map[int]string
type IMAPS = map[int]string

// 使用IMAPI替代map[int]int
type IMAPI = map[int]int

type MAPA = map[any]any

type Mapper interface {
	ToMap() map[string]any
}
