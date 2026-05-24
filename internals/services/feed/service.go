package feed

import (
	"context"

	"github.com/assaidy/vodzilla/internals/services"
)

// TODO: when implementing feed, account for soft-deleted users and videos
const Name = "feed"

var _ services.Service = (*Service)(nil)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }
