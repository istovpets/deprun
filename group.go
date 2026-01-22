// Package run implements an actor-runner with deterministic teardown. It is
// somewhat similar to package errgroup, except it does not require actor
// goroutines to understand context semantics. This makes it suitable for use in
// more circumstances; for example, goroutines which are handling connections
// from net.Listeners, or scanning input from a closable io.Reader.
package deprun

// Group collects actors (functions) and runs them concurrently.
// When one actor (function) returns, all actors are interrupted.
// The zero value of a Group is useful.
type Group struct {
	actors []actor
}

// AddDep adds a runnable that may resolve a dependency.
// The dependency is resolved only if ready is called.
//
// AddDep is used to add an actor that other actors can depend on.
// It returns a *Dependency object that can be passed to the `Add` method
// to create a dependency relationship. The actor added with AddDep must
// call the ready function to signal that it is ready and that dependent
// actors can start.
func (g *Group) AddDep(execute func(ready ReadySignal) error, interrupt func(error), dependsOn ...*Dependency) *Dependency {
	actor := actor{execute, interrupt, newDependency(), dependsOn}
	g.actors = append(g.actors, actor)

	return actor.provides
}

// Add an actor (function) to the group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
//
// To create a dependency, pass one or more *Dependency objects to Add.
// The actor will only start after all of its dependencies have signaled
// that they are ready. If no dependencies are provided, the actor starts
// immediately.
func (g *Group) Add(execute func() error, interrupt func(error), dependsOn ...*Dependency) {
	g.AddDep(func(ReadySignal) error { return execute() }, interrupt, dependsOn...)
}

// Run all actors (functions) concurrently.
// When the first actor returns, all others are interrupted.
// Run only returns when all actors have exited.
// Run returns the error returned by the first exiting actor.
func (g *Group) Run() error {
	if len(g.actors) == 0 {
		return nil
	}

	// Run each actor.
	errors := make(chan error, len(g.actors))
	for _, a := range g.actors {
		go func(a actor) {
			if !a.WaitDeps() {
				errors <- nil

				return // interrupted
			}

			errors <- a.execute(a.provides.ready)
		}(a)
	}

	// Wait for the first actor to stop.
	err := <-errors

	// Signal all actors to stop.
	for _, a := range g.actors {
		a.provides.interrupt()
		a.interrupt(err)
	}

	// Wait for all actors to stop.
	for i := 1; i < cap(errors); i++ {
		<-errors
	}

	// Return the original error.
	return err
}

type actor struct {
	execute   func(ready ReadySignal) error
	interrupt func(error)
	provides  *Dependency   // depend on me
	dependsOn []*Dependency // i'm dependent
}

func (a *actor) WaitDeps() bool {
	var interrupted bool
	for _, d := range a.dependsOn {
		if d == nil {
			continue
		}

		if !d.wait() {
			interrupted = true
		}
	}

	return !interrupted
}
