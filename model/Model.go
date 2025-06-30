package model

import (
	"context"
)

type Model struct {
	Ctx context.Context `json:"-"`
	Ext map[string]any  `json:"-"` //扩展属性
}

func (m *Model) WithContext(ctx context.Context) *Model {
	m.Ctx = ctx

	return m
}
