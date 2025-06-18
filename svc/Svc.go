package svc

import (
	"context"
)

type Svc struct {
	Ctx context.Context
}

func (s *Svc) WithContext(ctx context.Context) *Svc {
	s.Ctx = ctx

	return s
}
