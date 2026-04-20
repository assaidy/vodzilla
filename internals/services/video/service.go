package video

import (
	"context"

	"github.com/assaidy/video_streaming_app/internals/services"
)

// creat video safely:      Call User Service API + cache
// Avoid broken references:	Soft delete + catch `UserDeleted` event + background reconciliation

const Name = "video"

var _ services.Service = (*Service)(nil)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }
