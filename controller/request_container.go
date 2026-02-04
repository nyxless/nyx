package controller

import (
	"github.com/nyxless/nyx/x"
	"time"
)

var _, _ requestContainer = (*httpContainer)(nil), (*rpcContainer)(nil)

type requestContainer interface { // {{{
	// 获取所有参数
	GetParams() x.MAP
	// 获取参数, any 类型
	GetParam(key string) any
	// 获取string类型参数, 当 key 不存在时，使用默认值
	GetString(key string, defaultValues ...string) string
	// 获取Int类型参数, 当 key 不存在时，使用默认值
	GetInt(key string, defaultValues ...int) int
	// 获取Int8类型参数
	GetInt8(key string, defaultValues ...int8) int8
	// 获取Int16类型参数
	GetInt16(key string, defaultValues ...int16) int16
	// 获取Int32类型参数
	GetInt32(key string, defaultValues ...int32) int32
	// 获取Int64类型参数
	GetInt64(key string, defaultValues ...int64) int64
	// 获取bool类型参数
	GetBool(key string, defaultValues ...bool) bool
	// 获取float类型参数
	GetFloat(key string, defaultValues ...float64) float64
	// 获取float32类型参数
	GetFloat32(key string, defaultValues ...float32) float32
	// 获取float64类型参数
	GetFloat64(key string, defaultValues ...float64) float64
	// 获取json字符串并转换为MAP
	GetJsonMap(key string) x.MAP
	// 获取[]any 类型参数，若值为字符分隔的字符串, 会分割并转换成[]any
	GetSlice(key string, separators ...string) []any
	// 获取[]string 类型参数，若值为字符分隔的字符串, 会分割并转换成[]string
	GetStringSlice(key string, separators ...string) []string
	// 获取[]int 类型参数，若值为字符分隔的字符串, 会分割并转换成[]int
	GetIntSlice(key string, separators ...string) []int
	// 获取[]int32 类型参数，若值为字符分隔的字符串, 会分割并转换成[]int32
	GetInt32Slice(key string, separators ...string) []int32
	// 获取[]int64  类型参数，若值为字符分隔的字符串, 会分割并转换成[]int64
	GetInt64Slice(key string, separators ...string) []int64
	// 获取[]map[string]any类型参数, 或强制转换类型
	GetMapSlice(key string) []x.MAP
	// 获取bytes类型参数, 或强制转换类型
	GetBytes(key string, defaultValues ...[]byte) []byte
	// 获取map[string]any类型参数
	GetMap(key string) x.MAP
	// 获取map[string]string类型参数
	GetStringMap(key string) x.MAPS
	// 获取map[string]int类型参数
	GetIntMap(key string) x.MAPI
	// 获取time.Time类型参数
	GetTime(key string) time.Time

	GetIp() string

	GetHeader(key string, defaultValues ...string) (ret string)
	GetHeaders() (ret x.MAPS)
	SetHeader(key, val string)
	SetHeaders(headers x.MAPS)

	Render(data ...any)
	RenderError(err any)
	RenderStream(data any) error
} // }}}
