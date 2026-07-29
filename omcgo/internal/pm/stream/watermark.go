package stream

// QueueWatermarkAllowsClose is the timeout-close state machine. A time window
// may close only after its minimum grace has elapsed and every durable source
// event that belongs before the window boundary has been consumed.
func QueueWatermarkAllowsClose(graceElapsed bool, pendingBeforeBoundary int64) bool {
	return graceElapsed && pendingBeforeBoundary == 0
}
