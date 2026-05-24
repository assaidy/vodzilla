package social

import (
	"context"

	"github.com/assaidy/vodzilla/internals/services"
)

// TODO: when implementing social, account for soft-deleted users
const Name = "social"

var _ services.Service = (*Service)(nil)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }
