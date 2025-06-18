package model

import (
	"context"
)

type Model struct {
	Ctx context.Context `json:"-"`
}

func (m *Model) WithContext(ctx context.Context) *Model {
	m.Ctx = ctx

	return m
}
