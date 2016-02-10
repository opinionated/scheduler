package scheduler

import (
	"flag"
	"fmt"
	"github.com/opinionated/utils/log"
	"math/rand"
	"os"
	"testing"
	"time"
)

// Type used for testing the scheduler
type TestSchedulable struct {
	start time.Time
	delay float64
	msg   int
	c     chan int
}

// simple factory function
func MakeTestSchedulable(delay int, id int, i chan int) TestSchedulable {
	return TestSchedulable{time.Now(), float64(delay), id, i}
}

// double check we implement all the interfaces
var _ Schedulable = (*TestSchedulable)(nil)

// check if we are ready to run
func (schedulable TestSchedulable) TimeRemaining() int {
	// convert this to ms (doesn't have a ms for whatever reason
	del := schedulable.delay - time.Since(schedulable.start).Seconds()*1000
	if del < 500 {
		return 0
	}
	return int(del)
}

func (schedulable TestSchedulable) SetTimeRemaining(_ int) {}

func (schedulable TestSchedulable) ResetTimer() {
	schedulable.start = time.Now()
}

// run the task
func (schedulable TestSchedulable) Run(scheduler *Scheduler) {
	schedulable.c <- schedulable.msg
	//diff := schedulable.delay - time.Since(schedulable.start).Seconds()*1000
	since := time.Since(schedulable.start).Seconds() * 1000
	fmt.Println("doing ", schedulable.msg, "at time:", since)
}

func expectResponse(id int, runtimes chan int, t *testing.T) {
	select {
	case got := <-runtimes:
		if got != id {
			t.Errorf("expected: %d got: %d", id, got)
		}
	}
}

func expectNow(id int, runtimes chan int, t *testing.T) {
	select {
	case got := <-runtimes:
		if got != id {
			t.Errorf("expected: %d got: %d", id, got)
		}
	default:
		t.Errorf("task %d did not hit on time", id)
	}

}

func expectAfter(id int, runtimes chan int, t *testing.T, after int) {
	waiter := time.After(time.Duration(after) * time.Millisecond)
	select {
	case got := <-runtimes:
		if got != id {
			t.Errorf("expected: %d got: %d", id, got)
		}
	case <-waiter:
		t.Errorf("task %d took too long", id)
	}
}

func randomizeAdds(tasks []Schedulable, s *Scheduler) {
	for i := range rand.Perm(len(tasks)) {
		go s.Add(tasks[i])
	}
}

func expectEmpty(c chan int, t *testing.T) {
	select {
	case got := <-c:
		t.Errorf("expected empty but got: %d", got)
	default:

	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.InitStd()
	os.Exit(m.Run())
}

// check that the scheduler removes the tasks one by one properly
func TestSchedulerOneByOne(t *testing.T) {
	s := MakeScheduler(5, 3)
	s.SetCycleTime(1)

	runtimes := make(chan int)

	go s.Add(MakeTestSchedulable(1000, 1, runtimes))
	go s.Add(MakeTestSchedulable(2000, 2, runtimes))
	go s.Add(MakeTestSchedulable(3000, 3, runtimes))
	go s.Add(MakeTestSchedulable(4000, 4, runtimes))

	s.Start()

	ticker := time.NewTicker(1 * time.Second)
	for i := 0; i < 4; i++ {
		<-ticker.C
		expectResponse(i+1, runtimes, t)
	}

	close(runtimes)
}

func TestSchedulerPairs(t *testing.T) {
	s := MakeScheduler(5, 3)
	s.SetCycleTime(1)

	runtimes := make(chan int)

	go randomizeAdds([]Schedulable{
		MakeTestSchedulable(600, 1, runtimes),
		MakeTestSchedulable(2000, 2, runtimes),
		MakeTestSchedulable(3000, 3, runtimes),
		MakeTestSchedulable(4000, 4, runtimes),
	}, s)

	s.Start()

	expectAfter(1, runtimes, t, 1200)
	expectEmpty(runtimes, t)
	expectAfter(2, runtimes, t, 1100)
	expectEmpty(runtimes, t)
	expectAfter(3, runtimes, t, 1100)
	expectEmpty(runtimes, t)
	expectAfter(4, runtimes, t, 1100)
	expectEmpty(runtimes, t)

	close(runtimes)
}
