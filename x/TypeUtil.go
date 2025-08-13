package x

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// 强制转换为bool
func AsBool(b interface{}, def ...bool) bool { // {{{
	if val, ok := b.(bool); ok {
		return val
	}

	val, ok := b.(string)
	if !ok {
		val = fmt.Sprint(b)
	}

	ret, err := strconv.ParseBool(val)
	if nil != err {
		if len(def) > 0 {
			ret = def[0]
		}
	}

	return ret
} // }}}

// 强制转换为int
func AsInt(num interface{}, def ...int) int { // {{{
	numint := 0
	switch val := num.(type) {
	case int:
		numint = val
	case string:
		numint = ToInt(val)
	case int64:
		numint = int(val)
	case int32:
		numint = int(val)
	case float64:
		numint = int(val)
	case float32:
		numint = int(val)
	case uint:
		numint = int(val)
	case uint32:
		numint = int(val)
	case uint64:
		numint = int(val)
	default:
		numint = ToInt(fmt.Sprint(num))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为int32
func AsInt32(num interface{}, def ...int32) int32 { // {{{
	var numint int32
	switch val := num.(type) {
	case int32:
		numint = val
	case int:
		numint = int32(val)
	case string:
		numint = int32(ToInt(val))
	case int64:
		numint = int32(val)
	case float64:
		numint = int32(val)
	case float32:
		numint = int32(val)
	case uint:
		numint = int32(val)
	case uint32:
		numint = int32(val)
	case uint64:
		numint = int32(val)
	default:
		numint = int32(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为int64
func AsInt64(num interface{}, def ...int64) int64 { // {{{
	var numint int64
	switch val := num.(type) {
	case int64:
		numint = val
	case int:
		numint = int64(val)
	case string:
		numint = ToInt64(val)
	case int32:
		numint = int64(val)
	case float64:
		numint = int64(val)
	case float32:
		numint = int64(val)
	case uint:
		numint = int64(val)
	case uint32:
		numint = int64(val)
	case uint64:
		numint = int64(val)
	default:
		numint = ToInt64(fmt.Sprint(num))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint
func AsUint(num interface{}, def ...uint) uint { // {{{
	var numint uint
	switch val := num.(type) {
	case uint:
		numint = val
	case uint32:
		numint = uint(val)
	case uint64:
		numint = uint(val)
	case int:
		numint = uint(val)
	case string:
		numint = uint(ToInt(val))
	case int64:
		numint = uint(val)
	case int32:
		numint = uint(val)
	case float64:
		numint = uint(val)
	case float32:
		numint = uint(val)
	default:
		numint = uint(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint32
func AsUint32(num interface{}, def ...uint32) uint32 { // {{{
	var numint uint32
	switch val := num.(type) {
	case uint32:
		numint = val
	case uint:
		numint = uint32(val)
	case uint64:
		numint = uint32(val)
	case int:
		numint = uint32(val)
	case string:
		numint = uint32(ToInt(val))
	case int32:
		numint = uint32(val)
	case int64:
		numint = uint32(val)
	case float64:
		numint = uint32(val)
	case float32:
		numint = uint32(val)
	default:
		numint = uint32(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint64
func AsUint64(num interface{}, def ...uint64) uint64 { // {{{
	var numint uint64
	switch val := num.(type) {
	case uint64:
		numint = val
	case uint:
		numint = uint64(val)
	case uint32:
		numint = uint64(val)
	case int:
		numint = uint64(val)
	case string:
		numint = uint64(ToInt64(val))
	case int32:
		numint = uint64(val)
	case int64:
		numint = uint64(val)
	case float64:
		numint = uint64(val)
	case float32:
		numint = uint64(val)
	default:
		numint = uint64(ToInt64(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为float64
func AsFloat(num interface{}, def ...float64) float64 { // {{{
	return AsFloat64(num, def...)
} //}}}

// 强制转换为float32
func AsFloat32(num interface{}, def ...float32) float32 { // {{{
	var numint float32

	numint = float32(AsFloat64(num))

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为float64
func AsFloat64(num interface{}, def ...float64) float64 { // {{{
	var numint float64
	switch val := num.(type) {
	case float64:
		numint = val
	case int:
		numint = float64(val)
	case string:
		numint = ToFloat(val)
	case int32:
		numint = float64(val)
	case int64:
		numint = float64(val)
	case float32:
		numint = float64(val)
	default:
		numint = ToFloat(fmt.Sprint(num))
	}

	numint = ParseNaN(numint)

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
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
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprint(v)
	}
} // }}}

// 强制转换为bytes
func AsBytes(arg interface{}, def ...[]byte) []byte { // {{{
	if arg == nil {
		if len(def) > 0 {
			return def[0]
		}

		return nil
	}

	switch val := arg.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(AsString(arg))
	}
} // }}}

// string转换为int
func ToInt(num string) int { // {{{
	num = strings.TrimSpace(num)
	if num == "" {
		return 0
	}

	idx := strings.Index(num, ".")
	if idx > 0 {
		return int(ToFloat(num))
	}

	idx = strings.Index(num, "+")
	if idx > 0 && (num[idx-1] == 'e' || num[idx-1] == 'E') {
		return int(ToFloat(num))
	}

	numint, err := strconv.Atoi(num)
	if nil != err {
		return 0
	}
	return numint
} // }}}

// string转换为int32
func ToInt32(v string) int32 {
	return int32(ToInt(v))
}

// string转换为int64
func ToInt64(num string) int64 { // {{{
	num = strings.TrimSpace(num)
	if num == "" {
		return 0
	}

	numint, err := strconv.ParseInt(num, 10, 64)
	if nil != err {
		return 0
	}

	return numint
} // }}}

// NaN转换0
func ParseNaN(num float64) float64 { // {{{
	if math.IsNaN(num) || math.IsInf(num, 0) {
		num = 0
	}

	return num
} // }}}

// string转换为float64
func ToFloat(num string) float64 { // {{{
	return ToFloat64(num)
} // }}}

func ToFloat32(num string) float32 { // {{{
	return float32(ToFloat64(num))
} // }}}

func ToFloat64(num string) float64 { // {{{
	numfloat, err := strconv.ParseFloat(num, 64)
	if nil != err {
		return 0
	}

	return numfloat
} // }}}

// int转换为string
func ToString(num int) string { // {{{
	return strconv.Itoa(num)
} // }}}

// string的切片转换为int切片
func ToIntSlice(nums []string) []int { // {{{
	intnums := []int{}

	for _, v := range nums {
		intnums = append(intnums, ToInt(v))
	}

	return intnums
} // }}}

// string的切片转换为int32切片
func ToInt32Slice(nums []string) []int32 { // {{{
	intnums := []int32{}

	for _, v := range nums {
		intnums = append(intnums, int32(ToInt(v)))
	}

	return intnums
} // }}}

// string的切片转换为int64切片
func ToInt64Slice(nums []string) []int64 { // {{{
	intnums := []int64{}

	for _, v := range nums {
		intnums = append(intnums, ToInt64(v))
	}

	return intnums
} // }}}

// int的切片转换为string切片
func ToStringSlice(nums []int) []string { // {{{
	strnums := []string{}

	for _, v := range nums {
		strnums = append(strnums, strconv.Itoa(v))
	}

	return strnums
} // }}}

// interface{}的切片转换为int切片
func AsIntSlice(v interface{}) []int { // {{{
	if v == nil {
		return nil
	}

	var nums []int

	switch val := v.(type) {
	case []int:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt(i))
		}
	case []int32:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []int64:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为int32切片
func AsInt32Slice(v interface{}) []int32 { // {{{
	if v == nil {
		return nil
	}

	var nums []int32

	switch val := v.(type) {
	case []int32:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt32(i))
		}
	case []int:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []int64:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt32(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为int64切片
func AsInt64Slice(v interface{}) []int64 { // {{{
	if v == nil {
		return nil
	}

	var nums []int64

	switch val := v.(type) {
	case []int64:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt64(i))
		}
	case []int:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []int32:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt64(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为string切片
func AsStringSlice(v any) (strs []string) { // {{{
	if v == nil {
		return
	}

	switch val := v.(type) {
	case []string:
		return val
	case []any:
		for _, i := range val {
			strs = append(strs, AsString(i))
		}
	case [][]byte:
		for _, i := range val {
			strs = append(strs, string(i))
		}
	case []int:
		for _, i := range val {
			strs = append(strs, strconv.Itoa(i))
		}
	case []int8:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int16:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int32:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int64:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(i, 10))
		}
	case []uint:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint8:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint16:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint32:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint64:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(i, 10))
		}
	case []float32:
		for _, i := range val {
			strs = append(strs, strconv.FormatFloat(float64(i), 'f', -1, 32))
		}
	case []float64:
		for _, i := range val {
			strs = append(strs, strconv.FormatFloat(i, 'f', -1, 64))
		}
	case []bool:
		for _, i := range val {
			strs = append(strs, strconv.FormatBool(i))
		}
	case []fmt.Stringer:
		for _, i := range val {
			strs = append(strs, i.String())
		}

	default:
	}

	return
} // }}}

func AsSlice(v any) []any { // {{{
	if v == nil {
		return nil
	}

	var arr []any
	switch val := v.(type) {
	case []any:
		return val
	case []string:
		for _, i := range val {
			arr = append(arr, i)
		}
	case [][]byte:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []int:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []int8:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []int16:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []int32:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []int64:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []uint:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []uint8:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []uint16:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []uint32:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []uint64:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []float32:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []float64:
		for _, i := range val {
			arr = append(arr, i)
		}
	case []bool:
		for _, i := range val {
			arr = append(arr, i)
		}

	default:
	}

	return nil
} // }}}

func AsMap(a any) MAP { // {{{
	if m, ok := a.(MAP); ok {
		return m
	}

	if m, ok := a.(Mapper); ok {
		return m.ToMap()
	}

	data := MAP{}

	objVal := reflect.ValueOf(a)
	kind := objVal.Kind()

	if kind == reflect.Ptr {
		objVal = objVal.Elem()
		kind = objVal.Kind()
	}

	switch kind {
	case reflect.Map:
		keys := objVal.MapKeys()
		for _, k := range keys {
			data[AsString(k)] = objVal.MapIndex(k).Interface()
		}
	case reflect.Struct:
		t := objVal.Type()
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("json")
			if tag == "-" || tag == "nil" {
				continue
			}

			if tag == "" {
				tag = strings.ToLower(t.Field(i).Name)
			}

			data[tag] = objVal.Field(i).Interface()
		}
	}

	return data
} // }}}

func AsStringMap(a any) MAPS { // {{{
	m := AsMap(a)
	n := MAPS{}
	for k, v := range m {
		n[k] = AsString(v)
	}

	return n
} // }}}

func AsIntMap(a any) MAPI { // {{{
	m := AsMap(a)
	n := MAPI{}
	for k, v := range m {
		n[k] = AsInt(v)
	}

	return n
} // }}}

func AsMapSlice(a any) []MAP { // {{{
	n := []MAP{}
	if s, ok := a.([]any); ok {
		for _, v := range s {
			n = append(n, AsMap(v))
		}
	}

	return n
} // }}}
