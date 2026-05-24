// This binary runs cron jobs as a single worker instance.
// It shouldn't be part of the main app because multiple server instances may run concurrently,
// which would cause duplicate cron job execution.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/assaidy/workers"
)

func main() {
	wm := workers.NewWorkerManager()

	// TODO: register cleanup workers (cron jobs):
	//   - CleanupExpiredSessions: delete sessions WHERE expires_at < now()
	//   - CleanupExpiredEmailVerificationTokens: delete tokens WHERE expires_at < now()
	//   - CleanupExpiredUploads: scan Redis for expired upload_ttl keys,
	//     abort S3 multipart uploads, delete object_keys, publish UploadExpiredEvent

	wm.Start()
	defer wm.Stop()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
	<-quitChan
}
