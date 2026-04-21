// This binary runs cron jobs as a single worker instance.
// It shouldn't be part of the main app because multiple server instances may run concurrently,
// which would cause duplicate cron job execution.
package main

import "github.com/assaidy/workers"

func main() {
	wm := workers.NewWorkerManager()

	// register workers (ie. cron jobs) here...

	wm.Start()
}
