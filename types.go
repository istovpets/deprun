package deprun

import "sync"

// ReadySignal is a function that must be called by an actor to signal that
// it is ready. This will unblock any actors that depend on it.
type ReadySignal func()

// Dependency represents a dependency that an actor can have on another.
// It is a signaling mechanism that ensures an actor only starts after its
// dependencies are ready. A Dependency is returned by AddDep and can be
// passed to Add.
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

func (s *Dependency) wait() bool {
	<-s.ch

	return !s.interrupted
}

// ready resolves the dependency and unblocks dependents.
// It is optional: a dependency may never become ready.
func (s *Dependency) ready() {
	s.once.Do(func() {
		close(s.ch)
	})
}

func (s *Dependency) interrupt() {
	s.once.Do(func() {
		s.interrupted = true
		close(s.ch)
	})
}
