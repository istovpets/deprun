package deprun_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/istovpets/deprun"
)

func ExampleGroup_Add_basic() {
	var g deprun.Group
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			select {
			case <-time.After(time.Second):
				fmt.Printf("The first actor had its time elapsed\n")
				return nil
			case <-cancel:
				fmt.Printf("The first actor was canceled\n")
				return nil
			}
		}, func(err error) {
			fmt.Printf("The first actor was interrupted with: %v\n", err)
			close(cancel)
		})
	}
	{
		g.Add(func() error {
			fmt.Printf("The second actor is returning immediately\n")
			return errors.New("immediate teardown")
		}, func(err error) {
			// Note that this interrupt function is called, even though the
			// corresponding execute function has already returned.
			fmt.Printf("The second actor was interrupted with: %v\n", err)
		})
	}
	fmt.Printf("The group was terminated with: %v\n", g.Run())
	// Output:
	// The second actor is returning immediately
	// The first actor was interrupted with: immediate teardown
	// The second actor was interrupted with: immediate teardown
	// The first actor was canceled
	// The group was terminated with: immediate teardown
}

func ExampleGroup_Add_context() {
	ctx, cancel := context.WithCancel(context.Background())
	var g deprun.Group
	{
		ctx, cancel := context.WithCancel(ctx) // note: shadowed
		g.Add(func() error {
			return runUntilCanceled(ctx)
		}, func(error) {
			cancel()
		})
	}
	go cancel()
	fmt.Printf("The group was terminated with: %v\n", g.Run())
	// Output:
	// The group was terminated with: context canceled
}

func ExampleGroup_Add_listener() {
	var g deprun.Group
	{
		ln, _ := net.Listen("tcp", ":0")
		g.Add(func() error {
			defer fmt.Printf("http.Serve returned\n")
			return http.Serve(ln, http.NewServeMux())
		}, func(error) {
			ln.Close()
		})
	}
	{
		g.Add(func() error {
			return errors.New("immediate teardown")
		}, func(error) {
			//
		})
	}
	fmt.Printf("The group was terminated with: %v\n", g.Run())
	// Output:
	// http.Serve returned
	// The group was terminated with: immediate teardown
}

func runUntilCanceled(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func ExampleGroup_AddDep() {
	var g deprun.Group
	var dep *deprun.Dependency
	{
		dep = g.AddDep(func(ready deprun.ReadySignal) error {
			fmt.Println("first actor: running")

			ready()

			time.Sleep(500 * time.Millisecond) // stay alive

			return nil
		}, func(err error) {
			fmt.Printf("first actor: interrupted with %v\n", err)
		})
	}
	{
		g.Add(func() error {
			// second actor always runs after the first.
			fmt.Println("second actor: running, will fail")

			return errors.New("a failure")
		}, func(err error) {
			fmt.Printf("second actor: interrupted with %v\n", err)
		}, dep)
	}
	fmt.Printf("The group was terminated with: %v\n", g.Run())
	// Output:
	// first actor: running
	// second actor: running, will fail
	// first actor: interrupted with a failure
	// second actor: interrupted with a failure
	// The group was terminated with: a failure
}
