package beat

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Many tests schedule a job for every second, and then wait at most a second
// for it to run.  This amount is just slightly larger than 1 second to
// compensate for a few milliseconds of runtime.
const OneSecond = 1*time.Second + 50*time.Millisecond

func stop(c *Beat) chan bool {
	ch := make(chan bool)
	go func() {
		c.Stop()
		ch <- true
	}()
	return ch
}

// Start and stop beat with no jobs.
func TestNoJobs(t *testing.T) {
	beat := New()
	beat.Start()

	select {
	case <-time.After(OneSecond):
		t.Fatal("expected beat will be stopped immediately")
	case <-stop(beat):
	}
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}

// Start, stop, then add an job. Verify job doesn't run.
func TestStopCausesJobsToNotRun(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	beat := New()
	beat.Start()
	beat.Stop()
	beat.Add("* * * * * *", "TestStopCausesJobsToNotRun-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	select {
	case <-time.After(OneSecond):
		// No job ran!
	case <-wait(wg):
		t.Fatal("expected stopped beat does not run any job")
	}
}

// Add a job, start beat, expect it runs.
func TestAddBeforeRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	beat := New()
	beat.Add("* * * * * *", "TestAddBeforeRunning-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	beat.Start()
	defer beat.Stop()

	// Give beat 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond):
		t.Fatal("expected job runs")
	case <-wait(wg):
	}
}

// Start beat, add a job, expect it runs.
func TestAddWhileRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	beat := New()
	beat.Start()
	defer beat.Stop()
	beat.Add("* * * * * *", "TestAddWhileRunning-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	select {
	case <-time.After(OneSecond):
		t.Fatal("expected job runs")
	case <-wait(wg):
	}
}

// Adding a job after calling start results in multiple job invocations
func TestAddWhileRunningWithDelay(t *testing.T) {
	beat := New()
	beat.Start()
	defer beat.Stop()

	time.Sleep(5 * time.Second)

	var calls int64

	beat.Add("* * * * * *", "TestAddWhileRunningWithDelay-1",
		func(ctx context.Context, userdata any) { atomic.AddInt64(&calls, 1) },
		nil)

	<-time.After(OneSecond)
	if atomic.LoadInt64(&calls) != 1 {
		t.Errorf("called %d times, expected 1\n", calls)
	}
}

// Add a job, remove a job, start beat, expect nothing runs.
func TestRemoveBeforeRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	id := "TestRemoveBeforeRunning-1"

	beat := New()

	beat.Add("* * * * * *", id,
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	beat.Remove(id)
	beat.Start()
	defer beat.Stop()

	select {
	case <-time.After(OneSecond):
		// Success, shouldn't run
	case <-wait(wg):
		t.FailNow()
	}
}

// Start beat, add a job, remove it, expect it doesn't run.
func TestRemoveWhileRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	id := "TestRemoveWhileRunning-1"

	beat := New()
	beat.Start()
	defer beat.Stop()
	beat.Add("* * * * * *", id,
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	beat.Remove(id)

	select {
	case <-time.After(OneSecond):
	case <-wait(wg):
		t.FailNow()
	}
}

// Start beat, add a job, remove it by pattern, expect it doesn't run.
func TestRemoveByPattern(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	id := "TestRemoveByPattern-1"

	beat := New()
	beat.Start()
	defer beat.Stop()
	beat.Add("* * * * * *", id,
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	err := beat.RemoveByPattern("^TestRemoveByPattern-.*$")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(OneSecond):
	case <-wait(wg):
		t.FailNow()
	}
}

// Test that the jobs are correctly sorted.
// Add a bunch of long-in-the-future jobs, and an immediate job, and ensure
// that the immediate job runs immediately.
// Also: Test that multiple jobs run in the same instant.
func TestMultipleJobs(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	beat := New()
	beat.Add("1 1 * 0 0 0", "TestMultipleJobs-1",
		func(ctx context.Context, userdata any) {},
		nil)

	beat.Add("* * * * * *", "TestMultipleJobs-2",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Add("* * * * * *", "TestMultipleJobs-3",
		func(ctx context.Context, userdata any) { t.Fatal() },
		nil)

	beat.Add("* * * * * *", "TestMultipleJobs-4",
		func(ctx context.Context, userdata any) { t.Fatal() },
		nil)

	beat.Add("12 31 * 0 0 0",
		"TestMultipleJobs-5",
		func(ctx context.Context, userdata any) {},
		nil)

	beat.Add("* * * * * *", "TestMultipleJobs-6",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Remove("TestMultipleJobs-3")
	beat.Start()
	beat.Remove("TestMultipleJobs-4")
	defer beat.Stop()

	select {
	case <-time.After(OneSecond):
		t.Error("expected job run in proper order")
	case <-wait(wg):
	}
}

// Test running the same job twice.
func TestRunningJobTwice(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	beat := New()
	beat.Add("1 1 * 0 0 0", "TestRunningJobTwice-1",
		func(ctx context.Context, userdata any) {},
		nil)

	beat.Add("12 31 * 0 0 0", "TestRunningJobTwice-2",
		func(ctx context.Context, userdata any) {},
		nil)

	beat.Add("* * * * * *", "TestRunningJobTwice-3",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Start()
	defer beat.Stop()

	select {
	case <-time.After(2 * OneSecond):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

// Test that double-running is a no-op
func TestStartNoop(t *testing.T) {
	var tickChan = make(chan struct{}, 2)

	beat := New()

	beat.Add("* * * * * *", "TestStartNoop-1",
		func(ctx context.Context, userdata any) { userdata.(chan struct{}) <- struct{}{} },
		tickChan)

	beat.Start()
	defer beat.Stop()

	// Wait for the first firing to ensure the runner is going
	<-tickChan

	beat.Start()

	<-tickChan

	// Fail if this job fires again in a short period, indicating a double-run
	select {
	case <-time.After(time.Millisecond):
	case <-tickChan:
		t.Error("expected job fires exactly twice")
	}
}

func TestLocalTimezone(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	beat := New()

	tm := time.Now()
	if tm.Second() >= 58 {
		time.Sleep(2 * time.Second)
		tm = time.Now()
	}

	expr := fmt.Sprintf("%d %d %d %d %d %d,%d", tm.Month(), tm.Day(), tm.Weekday(), tm.Hour(), tm.Minute(), tm.Second()+1, tm.Second()+2)
	beat.Add(expr, "TestLocalTimezone-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Start()
	defer beat.Stop()

	// Give beat 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond * 2):
		t.Fatal("expected job runs", expr)
	case <-wait(wg):
	}
}

func TestNonLocalTimezone(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	loc, err := time.LoadLocation("Pacific/Majuro")
	if err != nil {
		panic(err)
	}

	beat := New(
		WithLocation(loc),
	)

	tm := time.Now().In(loc)
	if tm.Second() >= 58 {
		time.Sleep(2 * time.Second)
		tm = time.Now().In(loc)
	}

	expr := fmt.Sprintf("%d %d %d %d %d %d,%d",
		tm.Month(), tm.Day(), tm.Weekday(),
		tm.Hour(), tm.Minute(), tm.Second()+1, tm.Second()+2)

	beat.Add(expr, "TestNonLocalTimezone-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Start()
	defer beat.Stop()

	// Give beat 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond * 2):
		t.Fatal("expected job runs", expr)
	case <-wait(wg):
	}
}

func TestParserWithNonLocalTimezone(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	loc, err := time.LoadLocation("Pacific/Majuro")
	if err != nil {
		panic(err)
	}

	beat := New(
		WithParser(
			NewParser(
				WithDefaultLocation(loc),
			),
		),
	)

	tm := time.Now().In(loc)
	if tm.Second() >= 58 {
		time.Sleep(2 * time.Second)
		tm = time.Now().In(loc)
	}

	expr := fmt.Sprintf("%d %d %d %d %d %d,%d",
		tm.Month(), tm.Day(), tm.Weekday(),
		tm.Hour(), tm.Minute(), tm.Second()+1, tm.Second()+2)

	beat.Add(expr, "TestParserWithNonLocalTimezone-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	beat.Start()
	defer beat.Stop()

	// Give beat 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond * 2):
		t.Fatal("expected job runs", expr)
	case <-wait(wg):
	}
}

func TestRecovery(t *testing.T) {
	beat := New(WithRecovery())

	now := time.Now().Add(2 * time.Second)
	expr := fmt.Sprintf("%d %d %d %d %d %d",
		now.Month(), now.Day(), now.Weekday(),
		now.Hour(), now.Minute(), now.Second())

	beat.Add(expr, "TestRecovery", func(ctx context.Context, userdata any) {
		panic("panic in beat")
	}, nil)

	beat.Start()
	defer beat.Stop()

	time.Sleep(3 * time.Second)
}

func TestMaxGoroutines(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(3)

	beat := New(WithMaxGoroutines(2))

	now := time.Now().Add(1 * time.Second)
	expr := fmt.Sprintf("%d %d %d %d %d %d",
		now.Month(), now.Day(), now.Weekday(),
		now.Hour(), now.Minute(), now.Second())

	fn := func(ctx context.Context, userdata any) {
		time.Sleep(time.Second)
		wg.Done()
	}

	beat.Add(expr, "TestMaxGoroutines-1", fn, nil)
	beat.Add(expr, "TestMaxGoroutines-2", fn, nil)
	beat.Add(expr, "TestMaxGoroutines-3", fn, nil)

	beat.Start()
	defer beat.Stop()

	// Give beat 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond * 2):

	case <-wait(wg):
		t.Fatal("expected 2 jobs to run")
	}
}
