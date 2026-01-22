package deprun_test

import (
	"errors"
	"testing"
	"time"

	"github.com/istovpets/deprun"
)

func TestSingleDependency(t *testing.T) {
	const runs = 100

	for i := range runs {
		var group deprun.Group

		depReady := make(chan struct{})
		started := make(chan struct{})

		dependsOn := group.AddDep(
			func(ready deprun.ReadySignal) error {
				close(depReady)
				ready()

				<-started

				return nil
			},
			func(error) {},
		)

		group.Add(
			func() error {
				select {
				case <-depReady:
				default:
					t.Fatalf("run %d: dependency has not started", i)
				}

				close(started)

				return nil
			},
			func(error) {},
			dependsOn,
		)

		res := make(chan error, 1)
		go func() { res <- group.Run() }()

		select {
		case err := <-res:
			if err != nil {
				t.Fatalf("run %d: unexpected result error: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("run %d: test deadlocked", i)
		}
	}
}

func TestMultipleDependencies(t *testing.T) {
	const runs = 100

	for i := range runs {
		var group deprun.Group

		depAReady := make(chan struct{})
		depBReady := make(chan struct{})
		started := make(chan struct{})

		depA := group.AddDep(
			func(ready deprun.ReadySignal) error {
				close(depAReady)
				ready()

				<-started

				return nil
			},
			func(error) {},
		)

		depB := group.AddDep(
			func(ready deprun.ReadySignal) error {
				close(depBReady)
				ready()

				<-started

				return nil
			},
			func(error) {},
		)

		group.Add(
			func() error {
				select {
				case <-depAReady:
				default:
					t.Fatalf("run %d: dependency A has not started", i)
				}

				select {
				case <-depBReady:
				default:
					t.Fatalf("run %d: dependency B has not started", i)
				}

				close(started)

				return nil
			},
			func(error) {},
			depA, depB,
		)

		res := make(chan error, 1)
		go func() { res <- group.Run() }()

		select {
		case err := <-res:
			if err != nil {
				t.Fatalf("run %d: unexpected result error: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("run %d: test deadlocked", i)
		}
	}
}

func TestDependencyFailure(t *testing.T) {
	var group deprun.Group

	depErr := errors.New("dependency failed")

	dep := group.AddDep(
		func(ready deprun.ReadySignal) error {
			return depErr
		},
		func(error) {},
	)

	group.Add(
		func() error {
			t.Fatalf("dependent actor started despite dependency failure")

			return nil
		},
		func(error) {},
		dep,
	)

	res := make(chan error, 1)
	go func() { res <- group.Run() }()

	select {
	case err := <-res:
		if !errors.Is(err, depErr) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("group.Run deadlocked after dependency failure")
	}
}
