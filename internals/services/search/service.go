package search

import (
	"context"

	"github.com/assaidy/video_streaming_app/internals/services"
)

const Name = "search"

var _ services.Service = (*Service)(nil)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }
