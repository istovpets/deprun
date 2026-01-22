package deprun

import "sync"

type ReadySignal func()

type Dependency struct {
	once        sync.Once
	ch          chan struct{}
	interrupted bool
}

func newDependency() *Dependency {
	return &Dependency{
		ch: make(chan struct{}),
	}
}

func (s *Dependency) Wait() bool {
	<-s.ch

	return !s.interrupted
}

// Ready resolves the dependency and unblocks dependents.
// It is optional: a dependency may never become ready.
func (s *Dependency) Ready() {
	s.once.Do(func() {
		close(s.ch)
	})
}

func (s *Dependency) Interrupt() {
	s.once.Do(func() {
		s.interrupted = true
		close(s.ch)
	})
}
