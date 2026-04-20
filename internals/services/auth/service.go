package auth

import (
	"context"
	"database/sql"

	"github.com/assaidy/video_streaming_app/internals/services"
	"github.com/assaidy/video_streaming_app/internals/utils/mailer"
)

const Name = "auth"

var _ services.Service = (*Service)(nil)

type Service struct {
	db     *sql.DB
	mailer *mailer.Mailer
}

func New(db *sql.DB, mailer *mailer.Mailer) *Service {
	return &Service{db: db, mailer: mailer}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }
