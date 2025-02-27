package oobmigration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/derision-test/glock"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestRunner(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)
	refreshTicker := glock.NewMockTicker(time.Second * 30)

	store.ListFunc.SetDefaultReturn([]Migration{
		{ID: 1, Progress: 0.5},
	}, nil)

	runner := newRunner(store, refreshTicker, &observation.TestContext)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.SetDefaultReturn(0.5, nil)

	if err := runner.Register(1, migrator, MigratorOptions{ticker: ticker}); err != nil {
		t.Fatalf("unexpected error registering migrator: %s", err)
	}

	go runner.Start()
	tickN(ticker, 3)
	runner.Stop()

	if callCount := len(migrator.UpFunc.History()); callCount != 3 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 3, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 0 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 0, callCount)
	}
}

func TestRunnerError(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)
	refreshTicker := glock.NewMockTicker(time.Second * 30)

	store.ListFunc.SetDefaultReturn([]Migration{
		{ID: 1, Progress: 0.5},
	}, nil)

	runner := newRunner(store, refreshTicker, &observation.TestContext)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.SetDefaultReturn(0.5, nil)
	migrator.UpFunc.SetDefaultReturn(errors.New("uh-oh"))

	if err := runner.Register(1, migrator, MigratorOptions{ticker: ticker}); err != nil {
		t.Fatalf("unexpected error registering migrator: %s", err)
	}

	go runner.Start()
	tickN(ticker, 1)
	runner.Stop()

	if calls := store.AddErrorFunc.history; len(calls) != 1 {
		t.Fatalf("unexpected number of calls to AddError. want=%d have=%d", 1, len(calls))
	} else {
		if calls[0].Arg1 != 1 {
			t.Errorf("unexpected migrationId. want=%d have=%d", 1, calls[0].Arg1)
		}
		if calls[0].Arg2 != "uh-oh" {
			t.Errorf("unexpected error message. want=%s have=%s", "uh-oh", calls[0].Arg2)
		}
	}
}

func TestRunnerRemovesCompleted(t *testing.T) {
	store := NewMockStoreIface()
	ticker1 := glock.NewMockTicker(time.Second)
	ticker2 := glock.NewMockTicker(time.Second)
	ticker3 := glock.NewMockTicker(time.Second)
	refreshTicker := glock.NewMockTicker(time.Second * 30)

	store.ListFunc.SetDefaultReturn([]Migration{
		{ID: 1, Progress: 0.5},
		{ID: 2, Progress: 0.1, ApplyReverse: true},
		{ID: 3, Progress: 0.9},
	}, nil)

	runner := newRunner(store, refreshTicker, &observation.TestContext)

	// Makes no progress
	migrator1 := NewMockMigrator()
	migrator1.ProgressFunc.SetDefaultReturn(0.5, nil)

	// Goes to 0
	migrator2 := NewMockMigrator()
	migrator2.ProgressFunc.PushReturn(0.05, nil)
	migrator2.ProgressFunc.SetDefaultReturn(0, nil)

	// Goes to 1
	migrator3 := NewMockMigrator()
	migrator3.ProgressFunc.PushReturn(0.95, nil)
	migrator3.ProgressFunc.SetDefaultReturn(1, nil)

	if err := runner.Register(1, migrator1, MigratorOptions{ticker: ticker1}); err != nil {
		t.Fatalf("unexpected error registering migrator: %s", err)
	}
	if err := runner.Register(2, migrator2, MigratorOptions{ticker: ticker2}); err != nil {
		t.Fatalf("unexpected error registering migrator: %s", err)
	}
	if err := runner.Register(3, migrator3, MigratorOptions{ticker: ticker3}); err != nil {
		t.Fatalf("unexpected error registering migrator: %s", err)
	}

	go runner.Start()
	tickN(ticker1, 5)
	tickN(ticker2, 5)
	tickN(ticker3, 5)
	runner.Stop()

	// not finished
	if callCount := len(migrator1.UpFunc.History()); callCount != 5 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 5, callCount)
	}

	// finished after 2 updates
	if callCount := len(migrator2.DownFunc.History()); callCount != 1 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 1, callCount)
	}

	// finished after 2 updates
	if callCount := len(migrator3.UpFunc.History()); callCount != 1 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 1, callCount)
	}
}

func TestRunMigrator(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.SetDefaultReturn(0.5, nil)

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.5}
		tickN(ticker, 3)
	})

	if callCount := len(migrator.UpFunc.History()); callCount != 3 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 3, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 0 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 0, callCount)
	}
}

func TestRunMigratorMigrationErrors(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.SetDefaultReturn(0.5, nil)
	migrator.UpFunc.SetDefaultReturn(errors.New("uh-oh"))

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.5}
		tickN(ticker, 1)
	})

	if calls := store.AddErrorFunc.history; len(calls) != 1 {
		t.Fatalf("unexpected number of calls to AddError. want=%d have=%d", 1, len(calls))
	} else {
		if calls[0].Arg1 != 1 {
			t.Errorf("unexpected migrationId. want=%d have=%d", 1, calls[0].Arg1)
		}
		if calls[0].Arg2 != "uh-oh" {
			t.Errorf("unexpected error message. want=%s have=%s", "uh-oh", calls[0].Arg2)
		}
	}
}

func TestRunMigratorMigrationFinishesUp(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.PushReturn(0.8, nil)       // check
	migrator.ProgressFunc.PushReturn(0.9, nil)       // after up
	migrator.ProgressFunc.SetDefaultReturn(1.0, nil) // after up

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.8}
		tickN(ticker, 5)
	})

	if callCount := len(migrator.UpFunc.History()); callCount != 2 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 2, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 0 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 0, callCount)
	}
}

func TestRunMigratorMigrationFinishesDown(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.PushReturn(0.2, nil)       // check
	migrator.ProgressFunc.PushReturn(0.1, nil)       // after down
	migrator.ProgressFunc.SetDefaultReturn(0.0, nil) // after down

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.2, ApplyReverse: true}
		tickN(ticker, 5)
	})

	if callCount := len(migrator.UpFunc.History()); callCount != 0 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 0, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 2 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 2, callCount)
	}
}

func TestRunMigratorMigrationChangesDirection(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	migrator := NewMockMigrator()
	migrator.ProgressFunc.PushReturn(0.2, nil) // check
	migrator.ProgressFunc.PushReturn(0.1, nil) // after down
	migrator.ProgressFunc.PushReturn(0.0, nil) // after down
	migrator.ProgressFunc.PushReturn(0.0, nil) // re-check
	migrator.ProgressFunc.PushReturn(0.1, nil) // after up
	migrator.ProgressFunc.PushReturn(0.2, nil) // after up

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.2, ApplyReverse: true}
		tickN(ticker, 5)
		migrations <- Migration{ID: 1, Progress: 0.0, ApplyReverse: false}
		tickN(ticker, 5)
	})

	if callCount := len(migrator.UpFunc.History()); callCount != 5 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 5, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 2 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 2, callCount)
	}
}

func TestRunMigratorMigrationDesyncedFromData(t *testing.T) {
	store := NewMockStoreIface()
	ticker := glock.NewMockTicker(time.Second)

	progressValues := []float64{
		0.20,                         // inital check
		0.25, 0.30, 0.35, 0.40, 0.45, // after up (x5)
		0.45,                         // re-check
		0.50, 0.55, 0.60, 0.65, 0.70, // after up (x5)
	}

	migrator := NewMockMigrator()
	for _, val := range progressValues {
		migrator.ProgressFunc.PushReturn(val, nil)
	}

	runMigratorWrapped(store, migrator, ticker, func(migrations chan<- Migration) {
		migrations <- Migration{ID: 1, Progress: 0.2, ApplyReverse: false}
		tickN(ticker, 5)
		migrations <- Migration{ID: 1, Progress: 1.0, ApplyReverse: false}
		tickN(ticker, 5)
	})

	if callCount := len(migrator.UpFunc.History()); callCount != 10 {
		t.Errorf("unexpected number of calls to Up. want=%d have=%d", 10, callCount)
	}
	if callCount := len(migrator.DownFunc.History()); callCount != 0 {
		t.Errorf("unexpected number of calls to Down. want=%d have=%d", 0, callCount)
	}
}

// runMigratorWrapped creates a migrations channel, then passes it to both the runMigrator
// function and the given interact function, which execute concurrently. This channel can
// control the behavior of the migration controller from within the interact function.
//
// This method blocks until both functions return. The return of the interact function
// cancels a context controlling the runMigrator main loop.
func runMigratorWrapped(store storeIface, migrator Migrator, ticker glock.Ticker, interact func(migrations chan<- Migration)) {
	ctx, cancel := context.WithCancel(context.Background())
	migrations := make(chan Migration)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		runMigrator(
			ctx,
			store,
			migrator,
			migrations,
			migratorOptions{ticker: ticker},
			newOperations(&observation.TestContext),
		)
	}()

	interact(migrations)

	cancel()
	wg.Wait()
}

// tickN advances the given ticker by one second n times with a guaranteed reader.
func tickN(ticker *glock.MockTicker, n int) {
	for i := 0; i < n; i++ {
		ticker.BlockingAdvance(time.Second)
	}
}
