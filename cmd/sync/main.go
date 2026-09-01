package main

import (
	persistence "primotibalt/checkTests/topicpersistence"
)

func main() {
	var (
		breaker  = make(chan bool)
		notifier = make(chan bool)
	)

	// Routing first: without the tailnet rule nothing below can reach a partner.
	EnsureTailscaleRoute()

	// Catch up with the partners before the watcher starts, so the files we
	// pull in do not get broadcast straight back at them.
	Reconcile()

	AttachWatcher(persistence.QuestionsDir(), breaker)
	RunServer(breaker, notifier)
}
