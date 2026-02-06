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

func (c *Config) Get(keys ...string) *ConfVal { // {{{
	val, ok := GetMapNode(c.data, keys...)
	return &ConfVal{
		val, ok,
	}
} // }}}

func (c *Config) GetConifg(keys ...string) *Config { // {{{
	return &Config{
		c.Get(keys...).Map(),
	}
} // }}}

func (c *Config) GetString(keys ...string) string { // {{{
	return c.Get(keys...).String()
} // }}}

func (c *Config) GetInt(keys ...string) int { // {{{
	return c.Get(keys...).Int()
} // }}}

func (c *Config) GetInt64(keys ...string) int64 { // {{{
	return c.Get(keys...).Int64()
} // }}}

func (c *Config) GetBool(keys ...string) bool { // {{{
	return c.Get(keys...).Bool()
} // }}}

func (c *Config) GetMap(keys ...string) MAP { // {{{
	return c.Get(keys...).Map()
} // }}}

func (c *Config) GetStringMap(keys ...string) MAPS { // {{{
	return c.Get(keys...).StringMap()
} // }}}

func (c *Config) GetIntMap(keys ...string) MAPI { // {{{
	return c.Get(keys...).IntMap()
} // }}}

func (c *Config) GetSlice(keys ...string) []any { // {{{
	return c.Get(keys...).Slice()
} // }}}

func (c *Config) GetStringSlice(keys ...string) []string { // {{{
	return c.Get(keys...).StringSlice()
} // }}}

func (c *Config) GetIntSlice(keys ...string) []int { // {{{
	return c.Get(keys...).IntSlice()
} // }}}

func (c *Config) GetMapSlice(keys ...string) []MAP { // {{{
	return c.Get(keys...).MapSlice()
} // }}}

func (c *Config) GetMapsSlice(keys ...string) []MAPS { // {{{
	return c.Get(keys...).MapsSlice()
} // }}}

func (c *Config) GetDefString(def string, keys ...string) string { // {{{
	return c.Get(keys...).String(def)
} // }}}

func (c *Config) GetDefInt(def int, keys ...string) int { // {{{
	return c.Get(keys...).Int(def)
} // }}}

func (c *Config) GetDefInt64(def int64, keys ...string) int64 { // {{{
	return c.Get(keys...).Int64(def)
} // }}}

func (c *Config) GetDefBool(def bool, keys ...string) bool { // {{{
	return c.Get(keys...).Bool(def)
} // }}}

func (c *Config) GetDefMap(def MAP, keys ...string) MAP { // {{{
	return c.Get(keys...).Map(def)
} // }}}

func (c *Config) GetDefStringMap(def MAPS, keys ...string) MAPS { // {{{
	return c.Get(keys...).StringMap(def)
} // }}}

func (c *Config) GetDefIntMap(def MAPI, keys ...string) MAPI { // {{{
	return c.Get(keys...).IntMap(def)
} // }}}

func (c *Config) GetDefSlice(def []any, keys ...string) []any { // {{{
	return c.Get(keys...).Slice(def)
} // }}}

func (c *Config) GetDefStringSlice(def []string, keys ...string) []string { // {{{
	return c.Get(keys...).StringSlice(def)
} // }}}

func (c *Config) GetDefIntSlice(def []int, keys ...string) []int { // {{{
	return c.Get(keys...).IntSlice(def)
} // }}}

func (c *Config) GetDefMapSlice(def []MAP, keys ...string) []MAP { // {{{
	return c.Get(keys...).MapSlice(def)
} // }}}

func (c *Config) GetDefMapsSlice(def []MAPS, keys ...string) []MAPS { // {{{
	return c.Get(keys...).MapsSlice(def)
} // }}}
