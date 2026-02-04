package controller

import (
	"context"
	"github.com/nyxless/nyx/x"
	"net/url"
	"strings"
)

type CLI struct {
	Controller
	Form url.Values
}

func (c *CLI) Prepare(params url.Values, controller, action, group string) { // {{{
	c.Form = params
	c.Controller.Prepare(context.Background(), controller, action, group)
} // }}}

func (c *CLI) GetParams() x.MAP { // {{{
	params := x.MAP{}

	for k, v := range c.Form {
		if _, ok := params[k]; !ok && len(v) > 0 {
			params[k] = strings.TrimSpace(v[0])
		}
	}

	return params
} // }}}

func (c *CLI) GetParam(key string) any { // {{{
	if v := c.Form[key]; len(v) > 0 {
		return strings.TrimSpace(v[0])
	}

	return nil
} // }}}

func (c *CLI) GetString(key string, defaultValues ...string) string { // {{{
	if v, ok := c.Form[key]; ok && len(v) > 0 {
		return strings.TrimSpace(v[0])
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ""
} // }}}

func (c *CLI) GetInt(key string, defaultValues ...int) int { // {{{
	if v, ok := c.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}
