package x

import (
	"fmt"
	"github.com/nyxless/nyx/x/yaml"
	"path/filepath"
)

type Config struct {
	data map[string]any
}

func NewConfig(conf_files ...string) (*Config, string, error) { // {{{
	var conf_file string

	for _, v := range conf_files {
		if !filepath.IsAbs(v) {
			v = filepath.Join(AppRoot, v)
		}

		if isfile, _ := IsFile(v); isfile {
			conf_file = v
			break
		}
	}

	if conf_file == "" {
		return nil, "", fmt.Errorf("config file is not exists!")
	}

	y := yaml.NewYaml(conf_file)
	m, err := y.YamlToMap()
	if err != nil {
		return nil, conf_file, err
	}

	return &Config{m}, conf_file, nil
} // }}}

type ConfVal struct {
	val   any
	found bool
}

func (cv *ConfVal) String(def ...string) string { // {{{
	if cv.found {
		return AsString(cv.val)
	}

	var s string
	if len(def) > 0 {
		s = def[0]
	}

	return s
} // }}}

func (cv *ConfVal) Int(def ...int) int { // {{{
	if cv.found {
		return AsInt(cv.val)
	}

	var d int
	if len(def) > 0 {
		d = def[0]
	}

	return d
} // }}}

func (cv *ConfVal) Int64(def ...int64) int64 { // {{{
	if cv.found {
		return AsInt64(cv.val)
	}

	var d int64
	if len(def) > 0 {
		d = def[0]
	}

	return d
} // }}}

func (cv *ConfVal) Bool(def ...bool) bool { // {{{
	if cv.found {
		return AsBool(cv.val)
	}

	var b bool
	if len(def) > 0 {
		b = def[0]
	}

	return b
} // }}}

func (cv *ConfVal) Map(def ...MAP) MAP { // {{{
	if cv.found {
		return AsMap(cv.val)
	}

	var m MAP
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) StringMap(def ...MAPS) MAPS { // {{{
	if cv.found {
		return AsStringMap(cv.val)
	}

	var m MAPS
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) IntMap(def ...MAPI) MAPI { // {{{
	if cv.found {
		return AsIntMap(cv.val)
	}

	var m MAPI
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) Slice(def ...[]any) []any { // {{{
	if cv.found {
		return AsSlice(cv.val)
	}

	var m []any
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) StringSlice(def ...[]string) []string { // {{{
	if cv.found {
		return AsStringSlice(cv.val)
	}

	var m []string
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) IntSlice(def ...[]int) []int { // {{{
	if cv.found {
		return AsIntSlice(cv.val)
	}

	var m []int
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) MapSlice(def ...[]MAP) []MAP { // {{{
	if cv.found {
		if s, ok := cv.val.([]any); ok {
			result := []MAP{}
			for _, v := range s {
				if j, ok := v.(MAP); ok {
					result = append(result, j)
				} else {
					result = append(result, MAP{})
				}
			}

			return result
		}
	}

	var m []MAP
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (cv *ConfVal) MapsSlice(def ...[]MAPS) []MAPS { // {{{
	if cv.found {
		if s, ok := cv.val.([]any); ok {
			result := []MAPS{}
			for _, v := range s {
				result = append(result, AsStringMap(v))
			}

			return result
		}
	}

	var m []MAPS
	if len(def) > 0 {
		m = def[0]
	}

	return m
} // }}}

func (this *Config) Get(keys ...string) *ConfVal { // {{{
	val, ok := GetMapNode(this.data, keys...)
	return &ConfVal{
		val, ok,
	}
} // }}}

func (this *Config) GetConifg(keys ...string) *Config { // {{{
	return &Config{
		this.Get(keys...).Map(),
	}
} // }}}

func (this *Config) GetString(keys ...string) string { // {{{
	return this.Get(keys...).String()
} // }}}

func (this *Config) GetInt(keys ...string) int { // {{{
	return this.Get(keys...).Int()
} // }}}

func (this *Config) GetInt64(keys ...string) int64 { // {{{
	return this.Get(keys...).Int64()
} // }}}

func (this *Config) GetBool(keys ...string) bool { // {{{
	return this.Get(keys...).Bool()
} // }}}

func (this *Config) GetMap(keys ...string) MAP { // {{{
	return this.Get(keys...).Map()
} // }}}

func (this *Config) GetStringMap(keys ...string) MAPS { // {{{
	return this.Get(keys...).StringMap()
} // }}}

func (this *Config) GetIntMap(keys ...string) MAPI { // {{{
	return this.Get(keys...).IntMap()
} // }}}

func (this *Config) GetSlice(keys ...string) []any { // {{{
	return this.Get(keys...).Slice()
} // }}}

func (this *Config) GetStringSlice(keys ...string) []string { // {{{
	return this.Get(keys...).StringSlice()
} // }}}

func (this *Config) GetIntSlice(keys ...string) []int { // {{{
	return this.Get(keys...).IntSlice()
} // }}}

func (this *Config) GetMapSlice(keys ...string) []MAP { // {{{
	return this.Get(keys...).MapSlice()
} // }}}

func (this *Config) GetMapsSlice(keys ...string) []MAPS { // {{{
	return this.Get(keys...).MapsSlice()
} // }}}

func (this *Config) GetDefString(def string, keys ...string) string { // {{{
	return this.Get(keys...).String(def)
} // }}}

func (this *Config) GetDefInt(def int, keys ...string) int { // {{{
	return this.Get(keys...).Int(def)
} // }}}

func (this *Config) GetDefInt64(def int64, keys ...string) int64 { // {{{
	return this.Get(keys...).Int64(def)
} // }}}

func (this *Config) GetDefBool(def bool, keys ...string) bool { // {{{
	return this.Get(keys...).Bool(def)
} // }}}

func (this *Config) GetDefMap(def MAP, keys ...string) MAP { // {{{
	return this.Get(keys...).Map(def)
} // }}}

func (this *Config) GetDefStringMap(def MAPS, keys ...string) MAPS { // {{{
	return this.Get(keys...).StringMap(def)
} // }}}

func (this *Config) GetDefIntMap(def MAPI, keys ...string) MAPI { // {{{
	return this.Get(keys...).IntMap(def)
} // }}}

func (this *Config) GetDefSlice(def []any, keys ...string) []any { // {{{
	return this.Get(keys...).Slice(def)
} // }}}

func (this *Config) GetDefStringSlice(def []string, keys ...string) []string { // {{{
	return this.Get(keys...).StringSlice(def)
} // }}}

func (this *Config) GetDefIntSlice(def []int, keys ...string) []int { // {{{
	return this.Get(keys...).IntSlice(def)
} // }}}

func (this *Config) GetDefMapSlice(def []MAP, keys ...string) []MAP { // {{{
	return this.Get(keys...).MapSlice(def)
} // }}}

func (this *Config) GetDefMapsSlice(def []MAPS, keys ...string) []MAPS { // {{{
	return this.Get(keys...).MapsSlice(def)
} // }}}
