package x

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// 默认舍入类型
var DEFAULT_ROUND = RoundTrunc

type RoundType = int

const (
	RoundTrunc RoundType = iota // 截断小数
	RoundBank                   // 银行家舍入法（四舍六入五成双）
	RoundCeil                   // 向上取整
	RoundFloor                  // 向下取整
	RoundUp                     // 四舍五入
)

func round(x float64, roundType RoundType) float64 { // {{{
	switch roundType {
	case RoundUp:
		return math.Round(x)
	case RoundBank:
		return roundBank(x)
	case RoundCeil:
		return math.Ceil(x)
	case RoundFloor:
		return math.Floor(x)
	case RoundTrunc:
		return math.Trunc(x)
	default:
		return math.Round(x) // 默认四舍五入
	}
} // }}}

// 银行家舍入法（四舍六入五成双）
func roundBank(x float64) float64 { // {{{
	t := math.Trunc(x)
	d := math.Abs(x - t)

	// 判断是否为中间值（0.5）
	if d < 0.5 || (d == 0.5 && int64(t)%2 == 0) {
		return t
	}

	return t + math.Copysign(1, x)
} // }}}

func AsInt(num any, def ...int) int { //{{{
	return asInt(num, DEFAULT_ROUND, def...)
} // }}}

func AsInt8(num any, def ...int8) int8 { //{{{
	return asInt8(num, DEFAULT_ROUND, def...)
} // }}}

func AsInt16(num any, def ...int16) int16 { //{{{
	return asInt16(num, DEFAULT_ROUND, def...)
} // }}}

func AsInt32(num any, def ...int32) int32 { //{{{
	return asInt32(num, DEFAULT_ROUND, def...)
} // }}}

func AsInt64(num any, def ...int64) int64 { //{{{
	return asInt64(num, DEFAULT_ROUND, def...)
} // }}}

func AsUint(num any, def ...uint) uint { //{{{
	return asUint(num, DEFAULT_ROUND, def...)
} // }}}

func AsUint8(num any, def ...uint8) uint8 { //{{{
	return asUint8(num, DEFAULT_ROUND, def...)
} // }}}

func AsUint16(num any, def ...uint16) uint16 { //{{{
	return asUint16(num, DEFAULT_ROUND, def...)
} // }}}

func AsUint32(num any, def ...uint32) uint32 { //{{{
	return asUint32(num, DEFAULT_ROUND, def...)
} // }}}

func AsUint64(num any, def ...uint64) uint64 { //{{{
	return asUint64(num, DEFAULT_ROUND, def...)
} // }}}

func AsIntWithRound(num any, typ RoundType, def ...int) int { //{{{
	return asInt(num, typ, def...)
} // }}}

func AsInt8WithRound(num any, typ RoundType, def ...int8) int8 { //{{{
	return asInt8(num, typ, def...)
} // }}}

func AsInt16WithRound(num any, typ RoundType, def ...int16) int16 { //{{{
	return asInt16(num, typ, def...)
} // }}}

func AsInt32WithRound(num any, typ RoundType, def ...int32) int32 { //{{{
	return asInt32(num, typ, def...)
} // }}}

func AsInt64WithRound(num any, typ RoundType, def ...int64) int64 { //{{{
	return asInt64(num, typ, def...)
} // }}}

func AsUintWithRound(num any, typ RoundType, def ...uint) uint { //{{{
	return asUint(num, typ, def...)
} // }}}

func AsUint8WithRound(num any, typ RoundType, def ...uint8) uint8 { //{{{
	return asUint8(num, typ, def...)
} // }}}

func AsUint16WithRound(num any, typ RoundType, def ...uint16) uint16 { //{{{
	return asUint16(num, typ, def...)
} // }}}

func AsUint32WithRound(num any, typ RoundType, def ...uint32) uint32 { //{{{
	return asUint32(num, typ, def...)
} // }}}

func AsUint64WithRound(num any, typ RoundType, def ...uint64) uint64 { //{{{
	return asUint64(num, typ, def...)
} // }}}

// asInt 任意类型安全转换为 int
func asInt(num any, typ RoundType, def ...int) int { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		if val <= math.MaxInt {
			return int(val)
		}
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		if val <= math.MaxInt {
			return int(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= math.MinInt && f <= math.MaxInt {
			return int(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= math.MinInt && val <= math.MaxInt {
			return int(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r >= math.MinInt && r <= math.MaxInt {
			return int(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r >= math.MinInt && r <= math.MaxInt {
			return int(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= math.MinInt && v64 <= math.MaxInt {
			return int(v64)
		}
	case time.Time:
		return int(time.Now().Unix())
	default:
		if Debug {
			Noticef("[AsInt] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= math.MinInt && v64 <= math.MaxInt {
			return int(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asInt8 任意类型安全转换为 int8
func asInt8(num any, typ RoundType, def ...int8) int8 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case int8:
		return val
	case int:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val)
		}
	case int16:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val)
		}
	case int32:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val)
		}
	case int64:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val)
		}
	case uint:
		if val <= math.MaxInt8 {
			return int8(val)
		}
	case uint8:
		if val <= math.MaxInt8 {
			return int8(val)
		}
	case uint16:
		if val <= math.MaxInt8 {
			return int8(val)
		}
	case uint32:
		if val <= math.MaxInt8 {
			return int8(val)
		}
	case uint64:
		if val <= math.MaxInt8 {
			return int8(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= math.MinInt8 && f <= math.MaxInt8 {
			return int8(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r >= math.MinInt8 && r <= math.MaxInt8 {
			return int8(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r >= math.MinInt8 && r <= math.MaxInt8 {
			return int8(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= math.MinInt8 && v64 <= math.MaxInt8 {
			return int8(v64)
		}

	default:
		if Debug {
			Noticef("[AsInt8] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= math.MinInt8 && v64 <= math.MaxInt8 {
			return int8(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asInt16 任意类型安全转换为 int16
func asInt16(num any, typ RoundType, def ...int16) int16 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case int16:
		return val
	case int:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val)
		}
	case int8:
		return int16(val)
	case int32:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val)
		}
	case int64:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val)
		}
	case uint:
		if val <= math.MaxInt16 {
			return int16(val)
		}
	case uint8:
		return int16(val)
	case uint16:
		if val <= math.MaxInt16 {
			return int16(val)
		}
	case uint32:
		if val <= math.MaxInt16 {
			return int16(val)
		}
	case uint64:
		if val <= math.MaxInt16 {
			return int16(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= math.MinInt16 && f <= math.MaxInt16 {
			return int16(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r >= math.MinInt16 && r <= math.MaxInt16 {
			return int16(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r >= math.MinInt16 && r <= math.MaxInt16 {
			return int16(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= math.MinInt16 && v64 <= math.MaxInt16 {
			return int16(v64)
		}

	default:
		if Debug {
			Noticef("[AsInt16] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= math.MinInt16 && v64 <= math.MaxInt16 {
			return int16(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asInt32 任意类型安全转换为 int32
func asInt32(num any, typ RoundType, def ...int32) int32 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case int32:
		return val
	case int:
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			return int32(val)
		}
	case int8:
		return int32(val)
	case int16:
		return int32(val)
	case int64:
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			return int32(val)
		}
	case uint:
		if val <= math.MaxInt32 {
			return int32(val)
		}
	case uint8:
		return int32(val)
	case uint16:
		return int32(val)
	case uint32:
		if val <= math.MaxInt32 {
			return int32(val)
		}
	case uint64:
		if val <= math.MaxInt32 {
			return int32(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= math.MinInt32 && f <= math.MaxInt32 {
			return int32(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= math.MinInt32 && val <= math.MaxInt32 {
			return int32(round(val, typ))
		}

	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r >= math.MinInt32 && r <= math.MaxInt32 {
			return int32(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r >= math.MinInt32 && r <= math.MaxInt32 {
			return int32(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= math.MinInt32 && v64 <= math.MaxInt32 {
			return int32(v64)
		}

	default:
		if Debug {
			Noticef("[AsInt32] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= math.MinInt32 && v64 <= math.MaxInt32 {
			return int32(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asInt64 任意类型安全转换为 int64
func asInt64(num any, typ RoundType, def ...int64) int64 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= math.MinInt64 && f <= math.MaxInt64 {
			return int64(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= math.MinInt64 && val <= math.MaxInt64 {
			return int64(round(val, typ))
		}

	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r >= math.MinInt64 && r <= math.MaxInt64 {
			return int64(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r >= math.MinInt64 && r <= math.MaxInt64 {
			return int64(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		if v64, ok := stringToInt64(val, typ); ok {
			return v64
		}
	case time.Time:
		return time.Now().Unix()
	default:
		if Debug {
			Noticef("[AsInt64] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok {
			return v64
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asUint 任意类型安全转换为 uint
func asUint(num any, typ RoundType, def ...uint) uint { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case uint:
		return val
	case int:
		if val >= 0 {
			if v := uint(val); v <= math.MaxUint {
				return v
			}
		}
	case int8:
		if val >= 0 {
			return uint(val)
		}
	case int16:
		if val >= 0 {
			return uint(val)
		}
	case int32:
		if val >= 0 {
			return uint(val)
		}
	case int64:
		if val >= 0 {
			if v := uint(val); v <= math.MaxUint {
				return v
			}
		}
	case uint8:
		return uint(val)
	case uint16:
		return uint(val)
	case uint32:
		return uint(val)
	case uint64:
		if val <= math.MaxUint {
			return uint(val)
		}

	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= 0 && f <= math.MaxUint {
			return uint(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= 0 && val <= math.MaxUint {
			return uint(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r > 0 && r <= math.MaxUint {
			return uint(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r > 0 && r <= math.MaxUint {
			return uint(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		if v64, ok := stringToUint64(val, typ); ok && v64 <= math.MaxUint {
			return uint(v64)
		}
	case time.Time:
		return uint(time.Now().Unix())
	default:
		if Debug {
			Noticef("[AsUint] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToUint64(fmt.Sprint(val), typ); ok && v64 <= math.MaxUint {
			return uint(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asUint8 任意类型安全转换为 uint8
func asUint8(num any, typ RoundType, def ...uint8) uint8 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case uint8:
		return val
	case int:
		if val >= 0 && val <= math.MaxUint8 {
			return uint8(val)
		}
	case int8:
		if val >= 0 {
			return uint8(val)
		}
	case int16:
		if val >= 0 && val <= math.MaxUint8 {
			return uint8(val)
		}
	case int32:
		if val >= 0 && val <= math.MaxUint8 {
			return uint8(val)
		}
	case int64:
		if val >= 0 && val <= math.MaxUint8 {
			return uint8(val)
		}
	case uint:
		if val <= math.MaxUint8 {
			return uint8(val)
		}
	case uint16:
		if val <= math.MaxUint8 {
			return uint8(val)
		}
	case uint32:
		if val <= math.MaxUint8 {
			return uint8(val)
		}
	case uint64:
		if val <= math.MaxUint8 {
			return uint8(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= 0 && f <= math.MaxUint8 {
			return uint8(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= 0 && val <= math.MaxUint8 {
			return uint8(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r > 0 && r <= math.MaxUint8 {
			return uint8(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r > 0 && r <= math.MaxUint8 {
			return uint8(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= 0 && v64 <= math.MaxUint8 {
			return uint8(v64)
		}

	default:
		if Debug {
			Noticef("[AsUint8] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= 0 && v64 <= math.MaxUint8 {
			return uint8(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asUint16 任意类型安全转换为 uint16
func asUint16(num any, typ RoundType, def ...uint16) uint16 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case uint16:
		return val
	case int:
		if val >= 0 && val <= math.MaxUint16 {
			return uint16(val)
		}
	case int8:
		if val >= 0 {
			return uint16(val)
		}
	case int16:
		if val >= 0 {
			return uint16(val)
		}
	case int32:
		if val >= 0 && val <= math.MaxUint16 {
			return uint16(val)
		}
	case int64:
		if val >= 0 && val <= math.MaxUint16 {
			return uint16(val)
		}
	case uint:
		if val <= math.MaxUint16 {
			return uint16(val)
		}
	case uint8:
		return uint16(val)
	case uint32:
		if val <= math.MaxUint16 {
			return uint16(val)
		}
	case uint64:
		if val <= math.MaxUint16 {
			return uint16(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= 0 && f <= math.MaxUint16 {
			return uint16(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= 0 && val <= math.MaxUint16 {
			return uint16(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r > 0 && r <= math.MaxUint16 {
			return uint16(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r > 0 && r <= math.MaxUint16 {
			return uint16(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= 0 && v64 <= math.MaxUint16 {
			return uint16(v64)
		}

	default:
		if Debug {
			Noticef("[AsUint16] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= 0 && v64 <= math.MaxUint16 {
			return uint16(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asUint32 任意类型安全转换为 uint32
func asUint32(num any, typ RoundType, def ...uint32) uint32 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case uint32:
		return val
	case int:
		if val >= 0 && val <= math.MaxUint32 {
			return uint32(val)
		}
	case int8:
		if val >= 0 {
			return uint32(val)
		}
	case int16:
		if val >= 0 {
			return uint32(val)
		}
	case int32:
		if val >= 0 {
			return uint32(val)
		}
	case int64:
		if val >= 0 && val <= math.MaxUint32 {
			return uint32(val)
		}
	case uint:
		if val <= math.MaxUint32 {
			return uint32(val)
		}
	case uint8:
		return uint32(val)
	case uint16:
		return uint32(val)
	case uint64:
		if val <= math.MaxUint32 {
			return uint32(val)
		}
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= 0 && f <= math.MaxUint32 {
			return uint32(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= 0 && val <= math.MaxUint32 {
			return uint32(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r > 0 && r <= math.MaxUint32 {
			return uint32(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r > 0 && r <= math.MaxUint32 {
			return uint32(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if v64, ok := stringToInt64(val, typ); ok && v64 >= 0 && v64 <= math.MaxUint32 {
			return uint32(v64)
		}

	default:
		if Debug {
			Noticef("[AsUint32] unknown type %T: %v", num, num)
		}
		// 尝试转为字符串再解析
		if v64, ok := stringToInt64(fmt.Sprint(val), typ); ok && v64 >= 0 && v64 <= math.MaxUint32 {
			return uint32(v64)
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// asUint64 任意类型安全转换为 uint64
func asUint64(num any, typ RoundType, def ...uint64) uint64 { //{{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}

	switch val := num.(type) {
	case uint64:
		return val
	case int:
		if val >= 0 {
			return uint64(val)
		}
	case int8:
		if val >= 0 {
			return uint64(val)
		}
	case int16:
		if val >= 0 {
			return uint64(val)
		}
	case int32:
		if val >= 0 {
			return uint64(val)
		}
	case int64:
		if val >= 0 {
			return uint64(val)
		}
	case uint:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case float32:
		if f := float64(val); !math.IsNaN(f) && f >= 0 && f <= math.MaxUint64 {
			return uint64(round(f, typ))
		}
	case float64:
		if !math.IsNaN(val) && val >= 0 && val <= math.MaxUint64 {
			return uint64(round(val, typ))
		}
	case complex64:
		if r := float64(real(val)); !math.IsNaN(r) && r > 0 && r <= math.MaxUint64 {
			return uint64(round(r, typ))
		}
	case complex128:
		if r := real(val); !math.IsNaN(r) && r > 0 && r <= math.MaxUint64 {
			return uint64(round(r, typ))
		}
	case bool:
		if val {
			return 1
		}
		return 0

	case string:
		if f, ok := stringToUint64(val, typ); ok {
			return f
		}
	case time.Time:
		return uint64(time.Now().Unix())
	default:
		if Debug {
			Noticef("[AsUint64] unknown type %T: %v", num, num)
		}
		if f, ok := stringToUint64(fmt.Sprint(val), typ); ok {
			return f
		}
	}

	if len(def) > 0 {
		return def[0]
	}
	return 0
} // }}}

// 字符串转数值
//
//go:inline
func stringToInt64(s string, typ RoundType) (int64, bool) { //{{{
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// 先尝试整数
	if v, err := strconv.ParseInt(s, 0, 64); err == nil {
		return v, true
	}

	// 再尝试浮点数
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && f >= math.MinInt64 && f <= math.MaxInt64 {
		return int64(round(f, typ)), true
	}

	return 0, false
} // }}}

//go:inline
func stringToUint64(s string, typ RoundType) (uint64, bool) { //{{{
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// 先尝试整数
	if v, err := strconv.ParseUint(s, 0, 64); err == nil {
		return v, true
	}

	// 再尝试浮点数
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && f >= 0 && f <= math.MaxUint64 {
		return uint64(round(f, typ)), true
	}

	return 0, false
} // }}}

//go:inline
func ToFloat64(s string) (float64, bool) { // {{{
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		return f, true
	}

	return 0, false
} // }}}

func ToInt(s string) (int, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= math.MinInt && v64 <= math.MaxInt {
		return int(v64), true
	}
	return 0, false
} // }}}

func ToInt8(s string) (int8, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= math.MinInt8 && v64 <= math.MaxInt8 {
		return int8(v64), true
	}
	return 0, false
} // }}}

func ToInt16(s string) (int16, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= math.MinInt16 && v64 <= math.MaxInt16 {
		return int16(v64), true
	}
	return 0, false
} // }}}

func ToInt32(s string) (int32, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= math.MinInt32 && v64 <= math.MaxInt32 {
		return int32(v64), true
	}
	return 0, false
} // }}}

func ToInt64(s string) (int64, bool) {
	return stringToInt64(s, DEFAULT_ROUND)
}

func ToUint(s string) (uint, bool) { // {{{
	if v64, ok := stringToUint64(s, DEFAULT_ROUND); ok && v64 <= math.MaxUint {
		return uint(v64), true
	}
	return 0, false
} // }}}

func ToUint8(s string) (uint8, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= 0 && v64 <= math.MaxUint8 {
		return uint8(v64), true
	}
	return 0, false
} // }}}

func ToUint16(s string) (uint16, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= 0 && v64 <= math.MaxUint16 {
		return uint16(v64), true
	}
	return 0, false
} // }}}

func ToUint32(s string) (uint32, bool) { // {{{
	if v64, ok := stringToInt64(s, DEFAULT_ROUND); ok && v64 >= 0 && v64 <= math.MaxUint32 {
		return uint32(v64), true
	}
	return 0, false
} // }}}

func ToUint64(s string) (uint64, bool) {
	return stringToUint64(s, DEFAULT_ROUND)
}

// 强制转换为float64
func AsFloat(num any, def ...float64) float64 { // {{{
	return AsFloat64(num, def...)
} //}}}

// 强制转换为float32
func AsFloat32(num any, def ...float32) float32 { // {{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}

		return 0
	}

	return float32(AsFloat64(num))
} // }}}

// 强制转换为float64
func AsFloat64(num any, def ...float64) float64 { // {{{
	if num == nil {
		if len(def) > 0 {
			return def[0]
		}

		return 0
	}

	switch val := num.(type) {
	case float64:
		return ParseNaN(val)
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return ParseNaN(float64(val))
	case complex64:
		return ParseNaN(float64(real(val)))
	case complex128:
		return ParseNaN(real(val))
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		if f, ok := ToFloat64(val); ok {
			return f
		}
	default:
		if Debug {
			Noticef("[AsFloat64] using fmt.Sprint for type %T: %v", num, num)
		}

		if f, ok := ToFloat64(fmt.Sprint(num)); ok {
			return f
		}
	}

	if len(def) > 0 {
		return def[0]
	}

	return 0
} // }}}

// NaN转换0
func ParseNaN(num float64) float64 { // {{{
	if math.IsNaN(num) || math.IsInf(num, 0) {
		return 0
	}

	return num
} // }}}
