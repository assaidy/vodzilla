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

	// register workers (ie. cron jobs) here...

	wm.Start()
	defer wm.Stop()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
	<-quitChan
}
