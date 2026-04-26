package lint

// ProgressObserver receives lifecycle events from a directory walk
// so callers can render progress without coupling the processing
// layer to UI concerns. OnFile may be invoked concurrently by worker
// goroutines; OnStart and OnDone fire once on the calling goroutine.
type ProgressObserver interface {
	OnStart(total int)
	OnFile(path string)
	OnDone()
}

// nopObserver is a no-op ProgressObserver.
type nopObserver struct{}

func (nopObserver) OnStart(int)   {}
func (nopObserver) OnFile(string) {}
func (nopObserver) OnDone()       {}
