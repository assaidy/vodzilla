package history

import (
	"context"

	"github.com/assaidy/vodzilla/internals/services"
)

type Service interface {
	services.Service
}

type impl struct{}

func New() Service {
	return &impl{}
}

func (me *impl) Start(ctx context.Context) error { return nil }
func (me *impl) Stop(ctx context.Context) error  { return nil }
