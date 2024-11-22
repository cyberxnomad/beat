package cron

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

func stop(c *Cron) chan bool {
	ch := make(chan bool)
	go func() {
		c.Stop()
		ch <- true
	}()
	return ch
}

// Start and stop cron with no jobs.
func TestNoJobs(t *testing.T) {
	cron := New()
	cron.Start()

	select {
	case <-time.After(OneSecond):
		t.Fatal("expected cron will be stopped immediately")
	case <-stop(cron):
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

	cron := New()
	cron.Start()
	cron.Stop()
	cron.Add("* * * * * * *", "1", "TestStopCausesJobsToNotRun-1", func(ctx context.Context, userdata any) { wg.Done() }, nil)

	select {
	case <-time.After(OneSecond):
		// No job ran!
	case <-wait(wg):
		t.Fatal("expected stopped cron does not run any job")
	}
}

// Add a job, start cron, expect it runs.
func TestAddBeforeRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.Add("* * * * * * *",
		"1", "TestAddBeforeRunning-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond):
		t.Fatal("expected job runs")
	case <-wait(wg):
	}
}

// Start cron, add a job, expect it runs.
func TestAddWhileRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.Start()
	defer cron.Stop()
	cron.Add("* * * * * * *",
		"1", "TestAddWhileRunning-1",
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
	cron := New()
	cron.Start()
	defer cron.Stop()

	time.Sleep(5 * time.Second)

	var calls int64

	cron.Add("* * * * * * *",
		"1", "TestAddWhileRunningWithDelay-1",
		func(ctx context.Context, userdata any) { atomic.AddInt64(&calls, 1) },
		nil)

	<-time.After(OneSecond)
	if atomic.LoadInt64(&calls) != 1 {
		t.Errorf("called %d times, expected 1\n", calls)
	}
}

// Add a job, remove a job, start cron, expect nothing runs.
func TestRemoveBeforeRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	id := "1"

	cron := New()

	cron.Add("* * * * * * *",
		id, "TestRemoveBeforeRunning-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	cron.Remove(id)
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(OneSecond):
		// Success, shouldn't run
	case <-wait(wg):
		t.FailNow()
	}
}

// Start cron, add a job, remove it, expect it doesn't run.
func TestRemoveWhileRunning(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	id := "1"

	cron := New()
	cron.Start()
	defer cron.Stop()
	cron.Add("* * * * * * *",
		id, "TestRemoveWhileRunning-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)
	cron.Remove(id)

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

	cron := New()
	cron.Add("* 1 1 * 0 0 0",
		"1", "TestMultipleJobs-1",
		func(ctx context.Context, userdata any) {},
		nil)

	cron.Add("* * * * * * *",
		"2", "TestMultipleJobs-2",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	cron.Add("* * * * * * *",
		"3", "TestMultipleJobs-3",
		func(ctx context.Context, userdata any) { t.Fatal() },
		nil)

	cron.Add("* * * * * * *",
		"4", "TestMultipleJobs-4",
		func(ctx context.Context, userdata any) { t.Fatal() },
		nil)

	cron.Add("* 12 31 * 0 0 0",
		"5", "TestMultipleJobs-5",
		func(ctx context.Context, userdata any) {},
		nil)

	cron.Add("* * * * * * *",
		"6", "TestMultipleJobs-6",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	cron.Remove("3")
	cron.Start()
	cron.Remove("4")
	defer cron.Stop()

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

	cron := New()
	cron.Add("* 1 1 * 0 0 0",
		"1", "TestRunningJobTwice-1",
		func(ctx context.Context, userdata any) {},
		nil)

	cron.Add("* 12 31 * 0 0 0",
		"2", "TestRunningJobTwice-2",
		func(ctx context.Context, userdata any) {},
		nil)

	cron.Add("* * * * * * *",
		"3", "TestRunningJobTwice-3",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * OneSecond):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

// Test that double-running is a no-op
func TestStartNoop(t *testing.T) {
	var tickChan = make(chan struct{}, 2)

	cron := New()

	cron.Add("* * * * * * *",
		"1", "TestStartNoop-1",
		func(ctx context.Context, userdata any) { userdata.(chan struct{}) <- struct{}{} },
		tickChan)

	cron.Start()
	defer cron.Stop()

	// Wait for the first firing to ensure the runner is going
	<-tickChan

	cron.Start()

	<-tickChan

	// Fail if this job fires again in a short period, indicating a double-run
	select {
	case <-time.After(time.Millisecond):
	case <-tickChan:
		t.Error("expected job fires exactly twice")
	}
}

func TestTimeZone(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()

	tm := time.Now().Add(2 * time.Second)
	expr := fmt.Sprintf("%d %d %d %d %d %d %d", tm.Year(), tm.Month(), tm.Day(), tm.Weekday(), tm.Hour(), tm.Minute(), tm.Second())
	cron.Add(expr, "1", "TestTimeZone-1",
		func(ctx context.Context, userdata any) { wg.Done() },
		nil)

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(OneSecond):
		t.Fatal("expected job runs", expr)
	case <-wait(wg):
	}
}
