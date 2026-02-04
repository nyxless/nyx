package x

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 强制转换为bool
func AsBool(b any, def ...bool) bool { // {{{
	if b == nil {
		if len(def) > 0 {
			return def[0]
		}
		return false
	}

	switch val := b.(type) {
	case bool:
		return val
	case string:
		str := strings.ToLower(strings.TrimSpace(val))
		switch str {
		case "1", "true", "t", "yes", "y", "on", "enable", "enabled", "ok":
			return true
		case "0", "false", "f", "no", "n", "off", "disable", "disabled":
			return false
		default:
			if num, err := strconv.ParseFloat(str, 64); err == nil {
				return math.Abs(num) > 1e-9
			}
		}
	case []byte:
		return AsBool(string(val), def...)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, complex64, complex128:
		return AsFloat64(val) != 0
	}

	if len(def) > 0 {
		return def[0]
	}

	return false
} // }}}

// 强制转换为string
func AsString(v any, def ...string) string { // {{{
	if v == nil {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		if num := ParseNaN(float64(val)); num > 0 {
			return strconv.FormatFloat(num, 'f', -1, 32)
		}
	case float64:
		if ParseNaN(val) > 0 {
			return strconv.FormatFloat(val, 'f', -1, 64)
		}
	case bool:
		return strconv.FormatBool(val)
	case complex64:
		if ParseNaN(float64(real(val))) > 0 && ParseNaN(float64(imag(val))) > 0 {
			return strconv.FormatComplex(complex128(val), 'f', -1, 64)
		}
	case complex128:
		if ParseNaN(real(val)) > 0 && ParseNaN(imag(val)) > 0 {
			return strconv.FormatComplex(val, 'f', -1, 128)
		}
	case fmt.Stringer:
		return val.String()
	case []any:
		return SliceToString(val)
	case MAP:
		return MapToString(val)
	case error:
		return val.Error()
	default:
		if len(def) > 0 {
			return def[0]
		}

		if Debug {
			Noticef("[AsString] using fmt.Sprint for type %T: %v", v, v)
		}
		return fmt.Sprint(v)
	}

	return ""
} // }}}

// 强制转换为bytes
func AsBytes(v any, def ...[]byte) []byte { // {{{
	if val, ok := v.([]byte); ok {
		return val
	}

	val := AsString(v)
	if val == "" {
		if len(def) > 0 {
			return def[0]
		}

		return []byte{}
	}

	return []byte(val)
} // }}}

// string的切片转换为int切片
func ToIntSlice(nums []string) []int { // {{{
	if len(nums) == 0 {
		return []int{}
	}

	intnums := make([]int, len(nums))
	for i := range nums {
		intnums[i], _ = ToInt(nums[i])
	}

	return intnums
} // }}}

// string的切片转换为int32切片
func ToInt32Slice(nums []string) []int32 { // {{{
	if len(nums) == 0 {
		return []int32{}
	}

	intnums := make([]int32, len(nums))
	for i := range nums {
		intnums[i], _ = ToInt32(nums[i])
	}

	return intnums
} // }}}

// string的切片转换为int64切片
func ToInt64Slice(nums []string) []int64 { // {{{
	if len(nums) == 0 {
		return []int64{}
	}

	intnums := make([]int64, len(nums))
	for i := range nums {
		intnums[i], _ = ToInt64(nums[i])
	}

	return intnums
} // }}}

// int的切片转换为string切片
func ToStringSlice(nums []int) []string { // {{{
	if len(nums) == 0 {
		return []string{}
	}

	strs := make([]string, len(nums))
	for i := range nums {
		strs[i] = strconv.Itoa(nums[i])
	}

	return strs
} // }}}

// []any转String
func SliceToString(s []any) string { // {{{
	var sb strings.Builder
	for i := range s {
		if i > 0 {
			sb.WriteString(",")
		}

		sb.WriteString(AsString(s[i]))
	}

	return sb.String()
} // }}}

// 字符串格式化显示MAP
func MapToString(m MAP) string { // {{{
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(",")
		}

		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(AsString(m[k]))
	}
	return sb.String()
} // }}}

// any的切片转换为int切片
func AsIntSlice(v any, separators ...string) []int { // {{{
	if v == nil {
		return []int{}
	}

	var nums []int
	switch val := v.(type) {
	case []int:
		return val
	case []int8:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []int16:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []int32:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []int64:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []uint:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []uint8:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []uint16:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []uint32:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []uint64:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(val[i])
		}
	case []float32:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(ParseNaN(float64(val[i])))
		}
	case []float64:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(ParseNaN(val[i]))
		}
	case []bool:
		nums = make([]int, len(val))
		for i := range val {
			if val[i] {
				nums[i] = 1
			} else {
				nums[i] = 0
			}
		}
	case []complex64:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(ParseNaN(float64(real(val[i]))))
		}
	case []complex128:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = int(ParseNaN(real(val[i])))
		}
	case []string:
		nums = make([]int, len(val))
		for i := range val {
			nums[i], _ = ToInt(val[i])
		}
	case []any:
		nums = make([]int, len(val))
		for i := range val {
			nums[i] = AsInt(val[i])
		}
	case string:
		return SplitInt(val, separators...)
	default:
		if Debug {
			Noticef("[AsIntSlice] try to use reflect for type %T: %v", v, v)
		}

		rv := reflect.ValueOf(v)

		if !rv.IsValid() {
			return []int{}
		}

		kind := rv.Kind()

		for kind == reflect.Ptr {
			if rv.IsNil() {
				return []int{}
			}

			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []int{}
			}

			nums = make([]int, length)
			for i := 0; i < length; i++ {
				rvi := rv.Index(i)
				for rvi.Kind() == reflect.Ptr {
					rvi = rvi.Elem()
				}
				nums[i] = AsInt(rvi.Interface())
			}
		}
	}

	return nums
} // }}}

// any的切片转换为int32切片
func AsInt32Slice(v any, separators ...string) []int32 { // {{{
	if v == nil {
		return []int32{}
	}

	var nums []int32
	switch val := v.(type) {
	case []int32:
		return val

	case []int:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []int16:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}

	case []int64:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []uint:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []uint8:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []uint16:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []uint32:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []uint64:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(val[i])
		}
	case []float32:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(ParseNaN(float64(val[i])))
		}
	case []float64:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(ParseNaN(val[i]))
		}
	case []bool:
		nums = make([]int32, len(val))
		for i := range val {
			if val[i] {
				nums[i] = 1
			} else {
				nums[i] = 0
			}
		}
	case []complex64:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(ParseNaN(float64(real(val[i]))))
		}
	case []complex128:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = int32(ParseNaN(real(val[i])))
		}
	case []string:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i], _ = ToInt32(val[i])
		}
	case []any:
		nums = make([]int32, len(val))
		for i := range val {
			nums[i] = AsInt32(val[i])
		}

	case string:
		nums = SplitInt32(val, separators...)

	default:
		if Debug {
			Noticef("[AsInt32Slice] try to use reflect for type %T: %v", v, v)
		}

		rv := reflect.ValueOf(v)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []int32{}
			}

			nums = make([]int32, length)
			for i := 0; i < length; i++ {
				nums[i] = AsInt32(rv.Index(i).Interface())
			}
		}
	}

	return nums
} // }}}

// any的切片转换为int64切片
func AsInt64Slice(v any, separators ...string) []int64 { // {{{
	if v == nil {
		return []int64{}
	}

	var nums []int64
	switch val := v.(type) {
	case []int64:
		return val

	case []int:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []int16:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []int32:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []uint:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []uint8:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []uint16:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []uint32:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []uint64:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(val[i])
		}
	case []float32:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(ParseNaN(float64(val[i])))
		}
	case []float64:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(ParseNaN(val[i]))
		}
	case []bool:
		nums = make([]int64, len(val))
		for i := range val {
			if val[i] {
				nums[i] = 1
			} else {
				nums[i] = 0
			}
		}
	case []complex64:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(ParseNaN(float64(real(val[i]))))
		}
	case []complex128:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = int64(ParseNaN(real(val[i])))
		}
	case []string:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i], _ = ToInt64(val[i])
		}
	case []any:
		nums = make([]int64, len(val))
		for i := range val {
			nums[i] = AsInt64(val[i])
		}
	case string:
		nums = SplitInt64(val, separators...)
	default:
		if Debug {
			Noticef("[AsInt64Slice] try to use reflect for type %T: %v", v, v)
		}

		rv := reflect.ValueOf(v)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []int64{}
			}

			nums = make([]int64, length)
			for i := 0; i < length; i++ {
				nums[i] = AsInt64(rv.Index(i).Interface())
			}
		}
	}

	return nums
} // }}}

// any 类型的切片转换为 string 类型切片
func AsStringSlice(v any, separators ...string) []string { // {{{
	if v == nil {
		return []string{}
	}

	var strs []string
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = AsString(val[i])
		}
	case [][]byte:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = string(val[i])
		}
	case []int:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.Itoa(val[i])
		}
	case []int8:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatInt(int64(val[i]), 10)
		}
	case []int16:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatInt(int64(val[i]), 10)
		}
	case []int32:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatInt(int64(val[i]), 10)
		}
	case []int64:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatInt(val[i], 10)
		}
	case []uint:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatUint(uint64(val[i]), 10)
		}
	case []uint8:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatUint(uint64(val[i]), 10)
		}
	case []uint16:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatUint(uint64(val[i]), 10)
		}
	case []uint32:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatUint(uint64(val[i]), 10)
		}
	case []uint64:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatUint(val[i], 10)
		}
	case []float32:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatFloat(float64(val[i]), 'f', -1, 32)
		}
	case []float64:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatFloat(val[i], 'f', -1, 64)
		}
	case []bool:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = strconv.FormatBool(val[i])
		}
	case []complex64:
		strs = make([]string, len(val))
		for i := range val {
			if ParseNaN(float64(real(val[i]))) > 0 && ParseNaN(float64(imag(val[i]))) > 0 {
				strs[i] = strconv.FormatComplex(complex128(val[i]), 'f', -1, 64)
			} else {
				strs[i] = ""
			}
		}
	case []complex128:
		strs = make([]string, len(val))
		for i := range val {
			if ParseNaN(real(val[i])) > 0 && ParseNaN(imag(val[i])) > 0 {
				strs[i] = strconv.FormatComplex(val[i], 'f', -1, 128)
			} else {
				strs[i] = ""
			}
		}
	case []fmt.Stringer:
		strs = make([]string, len(val))
		for i := range val {
			strs[i] = val[i].String()
		}
	case string:
		return SplitString(val, separators...)
	default:
		if Debug {
			Noticef("[AsStringSlice] try to use reflect for type %T: %v", v, v)
		}

		rv := reflect.ValueOf(v)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []string{}
			}

			strs = make([]string, length)
			for i := 0; i < length; i++ {
				strs[i] = AsString(rv.Index(i).Interface())
			}
		}
	}

	return strs
} // }}}

func AsSlice(v any, separators ...string) []any { // {{{
	if v == nil {
		return []any{}
	}

	switch val := v.(type) {
	case []any:
		return val
	case []string:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case [][]byte:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []int:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []int8:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []int16:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []int32:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []int64:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []uint:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []uint8:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []uint16:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []uint32:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []uint64:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []float32:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []float64:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []bool:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []complex64:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []complex128:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case []time.Time:
		arr := make([]any, len(val))
		for i := range val {
			arr[i] = val[i]
		}
		return arr
	case string:
		return Split(val, separators...)
	default:
		if Debug {
			Noticef("[AsSlice] try to use reflect for type %T: %v", v, v)
		}

		rv := reflect.ValueOf(v)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []any{}
			}

			arr := make([]any, length)
			for i := 0; i < length; i++ {
				arr[i] = rv.Index(i).Interface()
			}
			return arr
		}

	}

	return []any{v}
} // }}}

func AsMap(a any) MAP { // {{{
	if a == nil {
		return MAP{}
	}

	switch val := a.(type) {
	case MAP:
		return val
	case MAPI:
		m := make(MAP, len(val))
		for k, v := range val {
			m[k] = v
		}
		return m
	case MAPS:
		m := make(MAP, len(val))
		for k, v := range val {
			m[k] = v
		}
		return m
	case IMAP:
		m := make(MAP, len(val))
		for k, v := range val {
			m[AsString(k)] = v
		}
		return m
	case IMAPS:
		m := make(MAP, len(val))
		for k, v := range val {
			m[AsString(k)] = v
		}
		return m
	case IMAPI:
		m := make(MAP, len(val))
		for k, v := range val {
			m[AsString(k)] = v
		}
		return m
	case AMAP:
		m := make(MAP, len(val))
		for k, v := range val {
			m[AsString(k)] = v
		}
		return m
	case Mapper:
		return val.Map()
	default: // {{{
		if Debug {
			Noticef("[AsMap] try to use reflect for type %T: %v", a, a)
		}

		rv := reflect.ValueOf(a)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		switch kind {
		case reflect.Map:
			keys := rv.MapKeys()
			m := make(MAP, len(keys))
			for _, k := range keys {
				m[AsString(k)] = rv.MapIndex(k).Interface()
			}
			return m
		case reflect.Struct:
			t := rv.Type()
			m := make(MAP, t.NumField())
			for i := 0; i < t.NumField(); i++ {
				tag := t.Field(i).Tag.Get("json")
				if tag == "-" || tag == "nil" {
					continue
				}

				omitempty := false
				if tag == "" {
					tag = strings.ToLower(t.Field(i).Name)
				} else {
					parts := strings.Split(tag, ",")
					if len(parts) > 0 {
						tag = parts[0]
					}

					for _, part := range parts {
						if part == "omitempty" {
							omitempty = true
						}
					}
				}
				v := rv.Field(i)
				if !omitempty || (v.IsValid() && !v.IsZero()) {
					m[tag] = v.Interface()
				}
			}
			return m
		}

		// }}}
	}

	return MAP{}
} // }}}

func AsStringMap(a any) MAPS { // {{{
	m := AsMap(a)
	if len(m) == 0 {
		return MAPS{}
	}

	n := make(MAPS, len(m))
	for k, v := range m {
		n[k] = AsString(v)
	}

	return n
} // }}}

func AsIntMap(a any) MAPI { // {{{
	m := AsMap(a)
	if len(m) == 0 {
		return MAPI{}
	}

	n := make(MAPI, len(m))
	for k, v := range m {
		n[k] = AsInt(v)
	}

	return n
} // }}}

func AsIMapS(a any) IMAPS { // {{{
	m := AsMap(a)
	if len(m) == 0 {
		return IMAPS{}
	}

	n := make(IMAPS, len(m))
	for k, v := range m {
		n[AsInt(k)] = AsString(v)
	}

	return n
} // }}}

func AsIMapI(a any) IMAPI { // {{{
	m := AsMap(a)
	if len(m) == 0 {
		return IMAPI{}
	}

	n := make(IMAPI, len(m))
	for k, v := range m {
		n[AsInt(k)] = AsInt(v)
	}

	return n
} // }}}

func AsMapSlice(a any) []MAP { // {{{
	if a == nil {
		return []MAP{}
	}

	switch val := a.(type) {
	case []MAP:
		return val
	case []any:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []MAPI:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []MAPS:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []IMAP:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []IMAPS:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []IMAPI:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []AMAP:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = AsMap(val[i])
		}
		return n
	case []Mapper:
		n := make([]MAP, len(val))
		for i := range val {
			n[i] = val[i].Map()
		}
		return n
	default:
		if Debug {
			Noticef("[AsMapSlice] try to use reflect for type %T: %v", a, a)
		}

		rv := reflect.ValueOf(a)
		kind := rv.Kind()

		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return []MAP{}
			}

			n := make([]MAP, length)
			for i := 0; i < length; i++ {
				n[i] = AsMap(rv.Index(i).Interface())
			}
			return n
		}

		// 如果传入的是单个 map，包装成切片
		if kind == reflect.Map || kind == reflect.Struct {
			return []MAP{AsMap(a)}
		}
	}

	return []MAP{}
} // }}}

// 分隔字符串为[]any
func Split(str string, separators ...string) []any { // {{{
	if str == "" {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	parts := strings.Split(str, separator)
	slice := make([]any, len(parts))
	for i := range parts {
		slice[i] = strings.TrimSpace(parts[i])
	}

	return slice
} // }}}

// 分隔字符串为[]string
func SplitString(str string, separators ...string) []string { // {{{
	if str == "" {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	parts := strings.Split(str, separator)
	slice := make([]string, len(parts))
	for i := range parts {
		slice[i] = strings.TrimSpace(parts[i])
	}

	return slice
} // }}}

// 分隔字符串为[]int
func SplitInt(str string, separators ...string) []int { // {{{
	if str == "" {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	nums := strings.Split(str, separator)
	slice := make([]int, len(nums))
	for i := range nums {
		slice[i], _ = ToInt(nums[i])
	}

	return slice
} // }}}

// 分隔字符串为[]int32
func SplitInt32(str string, separators ...string) []int32 { // {{{
	if str == "" {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	nums := strings.Split(str, separator)
	slice := make([]int32, len(nums))
	for i := range nums {
		slice[i], _ = ToInt32(nums[i])
	}

	return slice
} // }}}

// 分隔字符串为[]int64
func SplitInt64(str string, separators ...string) []int64 { // {{{
	if str == "" {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	nums := strings.Split(str, separator)
	slice := make([]int64, len(nums))
	for i := range nums {
		slice[i], _ = ToInt64(nums[i])
	}

	return slice
} // }}}

// array 转换 map[T]struct{}
func ArrayToMap[T comparable](s []T) map[T]struct{} { // {{{
	m := make(map[T]struct{})
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
} // }}}

// 获取map树的某个节点 m[k1][k2]...[kn]
func GetMapNode(m MAP, keys ...string) (n any, found bool) { // {{{
	if len(keys) == 0 {
		return m, true
	}

	current := m
	for i, key := range keys {
		value, exists := current[key]
		if !exists {
			return nil, false
		}

		// 如果是最后一个key，直接返回
		if i == len(keys)-1 {
			return value, true
		}

		nextMap, ok := value.(MAP)
		if !ok {
			return nil, false
		}
		current = nextMap
	}

	return nil, false
} // }}}

// 从interface{}树中获得一个节点, 失败返回nil
func GetNode(node any, keys ...string) (n any, found bool) { // {{{
	if len(keys) == 0 {
		return node, true
	}

	m, ok := node.(MAP)
	if !ok {
		return nil, false
	}

	return GetMapNode(m, keys...)
} // }}}

// 从interface{}树中获得一个MAP类型, found 表示 key 是否存在, 存在但是类型不符，会转换成 MAP
func GetMap(node any, keys ...string) (m MAP, found bool) { // {{{
	if result, ok := GetNode(node, keys...); ok {
		return AsMap(result), true
	}

	return nil, false
} // }}}

// 从interface{}树中获得一个[]any类型, found 表示 key 是否存在, 存在但是类型不符，会转换成 []any
func GetSlice(node any, keys ...string) (s []any, found bool) { // {{{
	if result, ok := GetNode(node, keys...); ok {
		return AsSlice(result), true
	}

	return nil, false
} // }}}

// 合并MAP(一级)
func MapMerge[T comparable, K any](m map[T]K, ms ...map[T]K) map[T]K { // {{{
	for _, v := range ms {
		for i, j := range v {
			m[i] = j
		}
	}

	return m
} // }}}

func ArrayColumn[T comparable](m []map[string]T, column string) []T { // {{{
	if len(m) == 0 {
		return []T{}
	}

	n := make([]T, len(m))
	for i := range m {
		n[i] = m[i][column]
	}

	return n
} //}}}

func ArrayColumnUniq[T comparable](m []map[string]T, column string) []T { // {{{
	if len(m) == 0 {
		return []T{}
	}

	n := make([]T, 0, len(m))
	u := make(map[T]struct{}, len(m))
	for i := range m {
		k := m[i][column]
		if _, ok := u[k]; !ok {
			n = append(n, k)
			u[k] = struct{}{}
		}
	}

	return n
} //}}}

func ArrayColumnMap[T comparable](m []map[string]T, key, val string) map[T]T { // {{{
	n := map[T]T{}
	for _, i := range m {
		n[i[key]] = i[val]
	}

	return n
} //}}}

// 判断array/slice中是否存在某值
func InArray[T comparable](search T, s []T) bool { // {{{
	for _, v := range s {
		if search == v {
			return true
		}
	}

	return false
} // }}}

// 判断map中是否存在某值
func InMap[T, K comparable](search T, m map[K]T) bool { // {{{
	for _, v := range m {
		if search == v {
			return true
		}
	}

	return false
} // }}}

// array 去重
func ArrayUnique[T comparable](arr []T) []T { // {{{
	check_uniq := make(map[T]struct{}, len(arr))
	narr := make([]T, 0, len(arr))
	for i := range arr {
		if _, ok := check_uniq[arr[i]]; !ok {
			narr = append(narr, arr[i])
			check_uniq[arr[i]] = struct{}{}
		}
	}

	return narr
} // }}}

// array 新增，并去重
func ArrayMerge[T comparable](arr []T, n ...[]T) []T { // {{{
	total := len(arr)
	for _, v := range n {
		total += len(v)
	}

	if total == 0 {
		return []T{}
	}

	check_uniq := make(map[T]struct{}, total)
	result := make([]T, 0, total)
	for i := range arr {
		if _, ok := check_uniq[arr[i]]; !ok {
			result = append(result, arr[i])
			check_uniq[arr[i]] = struct{}{}
		}
	}

	for i := range n {
		for j := range n[i] {
			if _, ok := check_uniq[n[i][j]]; !ok {
				result = append(result, n[i][j])
				check_uniq[n[i][j]] = struct{}{}
			}
		}
	}

	return result
} // }}}

// array 删除
func ArrayRem[T comparable](arr []T, n T) []T { // {{{
	if len(arr) == 0 {
		return []T{}
	}

	narr := make([]T, 0, len(arr))
	for i := range arr {
		if arr[i] != n {
			narr = append(narr, arr[i])
		}
	}

	return narr
} // }}}

// 收集 map 的 key 到数组
func MapKeys[K comparable, V any](m map[K]V) []K { // {{{
	s := make([]K, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
} // }}}

// 收集 map 的 value 到数组
func MapValues[K comparable, V any](m map[K]V) []V { // {{{
	s := make([]V, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
} // }}}

// 交换 map 的 key/value
func MapReverse[K comparable, V comparable](m map[K]V) map[V]K { // {{{
	n := make(map[V]K, len(m))
	for k, v := range m {
		n[v] = k
	}
	return n
} // }}}

//
