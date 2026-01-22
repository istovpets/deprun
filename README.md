# deprun

[![Go Reference](https://pkg.go.dev/badge/github.com/istovpets/deprun.svg)](https://pkg.go.dev/github.com/istovpets/deprun)
[![Portfolio](https://img.shields.io/badge/author-portfolio-blue)](https://programmer.stovpets.com/)


`deprun` is an extension of the excellent [oklog/run](https://github.com/oklog/run) package. It provides a simple mechanism to manage dependencies between actors, ensuring that an actor only starts after one or more other actors have signaled that they are ready.

This is particularly useful in scenarios where you have a component (e.g., a web server, a database connector) that needs to wait for other components (e.g., a configuration loader, a metrics service) to be fully initialized before it can start.

## Installation

```bash
go get github.com/istovpets/deprun
```

## Usage

The core of `deprun` is the `Group` type. You can add actors and define dependencies between them.

- `AddDep(execute, interrupt)`: A convenience method to add an actor that other actors can depend on. It returns a `*Dependency` object.
- `Add(execute, interrupt, dependencies...)`: Adds an actor that will only start after all specified `*Dependency` objects have been signaled as ready. If the *Dependency object array is empty, the actor will run immediately.

### Example: Single Dependency

Here is a simple example demonstrating how to make one actor dependent on another. A "dependent" actor will only start after its "dependency" actor has called the `ready()` signal.

```go
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/istovpets/deprun"
)

func main() {
	var g deprun.Group

	// Actor 1: The dependency
	// This actor will run, signal it's ready, and then wait.
	dep := g.AddDep(func(ready deprun.ReadySignal) error {
		fmt.Println("dependency actor: running")
		time.Sleep(100 * time.Millisecond) // Simulate startup work
		ready() // Signal readiness
		fmt.Println("dependency actor: signaled ready")
		select {} // Wait for interruption
	}, func(err error) {
		fmt.Printf("dependency actor: interrupted with %v\n", err)
	})

	// Actor 2: The dependent
	// This actor depends on `dep` and will only start after `ready()` is called.
	g.Add(func() error {
		fmt.Println("dependent actor: running")
		time.Sleep(500 * time.Millisecond)
		return errors.New("a failure")
	}, func(err error) {
		fmt.Printf("dependent actor: interrupted with %v\n", err)
	}, dep) // Pass the dependency here

	// Actor 3: Signal handler
	cancelInterrupt := make(chan struct{})
	g.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return fmt.Errorf("received signal %s", sig)
		case <-cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})

	fmt.Printf("The group was terminated with: %v\n", g.Run())
}
```

Running this will produce output showing the dependent actor starting only after the dependency is ready.

### Example: Multiple Dependencies

An actor can depend on multiple other actors. It will only start after *all* of its dependencies have signaled that they are ready.

```go
package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/istovpets/deprun"
)

func main() {
	var g deprun.Group
	var wg sync.WaitGroup
	wg.Add(1)

	// Dependency A: A database connection
	dbDep := g.AddDep(func(ready deprun.ReadySignal) error {
		fmt.Println("database: connecting...")
		time.Sleep(100 * time.Millisecond)
		ready()
		fmt.Println("database: connected.")
		wg.Wait() // Keep running
		return nil
	}, func(err error) {
		fmt.Println("database: disconnected.")
	})

	// Dependency B: A metrics service
	metricsDep := g.AddDep(func(ready deprun.ReadySignal) error {
		fmt.Println("metrics: starting...")
		time.Sleep(150 * time.Millisecond)
		ready()
		fmt.Println("metrics: started.")
		wg.Wait() // Keep running
		return nil
	}, func(err error) {
		fmt.Println("metrics: stopped.")
	})

	// Dependent Actor: A web server
	// This actor depends on both the database and the metrics service.
	g.Add(func() error {
		fmt.Println("web server: starting...")
		// Now we can safely use the database and metrics
		fmt.Println("web server: started and ready to serve requests.")
		time.Sleep(500 * time.Millisecond)
		return errors.New("internal server error")
	}, func(err error) {
		fmt.Printf("web server: stopped due to %v.\n", err)
		wg.Done() // Signal other actors to stop
	}, dbDep, metricsDep) // Pass multiple dependencies

	fmt.Printf("The group was terminated with: %v\n", g.Run())
}
```

#### Output:
```
database: connecting...
metrics: starting...
database: connected.
metrics: started.
web server: starting...
web server: started and ready to serve requests.
web server: stopped due to internal server error.
database: disconnected.
metrics: stopped.
The group was terminated with: internal server error
```
As you can see, the "web server" only starts after both "database" and "metrics" have signaled they are ready.

## How it works

- **`Group.AddDep(execute, interrupt)`**: This is a convenience method that adds an actor to the group and returns a `*deprun.Dependency` object. This object can then be passed to other actors.
- **`Group.Add(execute, interrupt, dependencies...)`**: This is the extended `Add` method. You can pass one or more `*deprun.Dependency` objects. The `execute` function for this actor will not be called until **all** of its dependencies have signaled they are ready.
- **`deprun.ReadySignal`**: This is a function (`func()`) passed to the `execute` function of an actor that others depend on. The actor must call this function to signal that it has successfully initialized and other actors can now start.

## Original Project

This project is built upon and inspired by `oklog/run`. For more advanced usage and a deeper understanding of the actor model, please refer to the [original `oklog/run` repository](https://github.com/oklog/run).
