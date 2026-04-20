package cron

import "github.com/assaidy/workers"

func main() {
	// TODO: arcive workers repo and move it to one with a new name: `cron`
	wm := workers.NewWorkerManager()

	// register workers (ie. cron jobs) here...

	wm.Start()
}
