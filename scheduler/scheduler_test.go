package scheduler

import "testing"
import "time"
import "fmt"

// Type used for testing the scheduler
type TestSchedulable struct {
	start time.Time
	delay float64
	msg   string
	c     chan string
}

// simple factory function
func MakeTestSchedulable(delay int, id string, i chan string) TestSchedulable {
	return TestSchedulable{time.Now(), float64(delay), id, i}
}

// double check we implement all the interfaces
var _ Schedulable = (*TestSchedulable)(nil)

// check if we are ready to run
func (schedulable TestSchedulable) TimeRemaining() int {
	// convert this to ms
	del := schedulable.delay - time.Since(schedulable.start).Seconds()*1000
	if del < 500 {
		return 0
	}
	return int(del)
}

func (schedulable TestSchedulable) SetTimeRemaining(_ int) {}

func (schedulable TestSchedulable) IsLoopable() bool {
	return false
}

func (schedulable TestSchedulable) ResetTimer() {
	schedulable.start = time.Now()
}

// run the task
func (schedulable TestSchedulable) Run(scheduler *Scheduler) {
	schedulable.c <- schedulable.msg
	diff := schedulable.delay - time.Since(schedulable.start).Seconds()*1000
	fmt.Println("doing ", schedulable.msg, "at diff:", diff)
}

// check that the scheduler is threadsafe, runs tasks in the right order
func TestSchedulerSimple(t *testing.T) {

	s := MakeScheduler(5, 3)
	s.SetCycleTime(1)
	runtimes := make(chan string)
	go s.Add(MakeTestSchedulable(3300, "d", runtimes))
	go s.Add(MakeTestSchedulable(3200, "c", runtimes))
	go s.Add(MakeTestSchedulable(1000, "a", runtimes))
	go s.Add(MakeTestSchedulable(3100, "b", runtimes))
	go s.Add(MakeTestSchedulable(4400, "e", runtimes))
	go s.Add(MakeTestSchedulable(4700, "f", runtimes))

	s.Start()
	// wait until all the waiting tasks run
	time.Sleep(1200 * time.Millisecond)

	expectResponse := func(id string) {
		select {
		case got := <-runtimes:
			if got != id {
				t.Errorf("expected: %s got: %s", id, got)
			}
		default:
			t.Errorf("%s not ready", id)
		}
	}

	expectNoResponse := func() {
		select {
		case got := <-runtimes:
			t.Errorf("Expected nothing but found %s", got)
		default:
		}
	}

	expectResponse("a")
	expectNoResponse()

	time.Sleep(2000 * time.Millisecond)
	expectResponse("b")
	expectResponse("c")
	expectResponse("d")
	expectNoResponse()

	time.Sleep(1000 * time.Millisecond)
	expectResponse("e")
	expectNoResponse()

	time.Sleep(1000 * time.Millisecond)
	expectResponse("f")
	expectNoResponse()

	close(runtimes)
	// tell the scheduler we are all done
	s.Stop()
}

func TestSchedulerTight(t *testing.T) {

	s := MakeScheduler(5, 3)
	s.SetCycleTime(1)

	s.Start()

	runtimes := make(chan string)
	go s.Add(MakeTestSchedulable(2000, "b", runtimes))
	go s.Add(MakeTestSchedulable(3100, "c", runtimes))
	go s.Add(MakeTestSchedulable(1000, "a", runtimes))
	go s.Add(MakeTestSchedulable(3400, "d", runtimes))
	go s.Add(MakeTestSchedulable(4100, "e", runtimes))

	first := time.After(1100 * time.Millisecond)
	second := time.After(2100 * time.Millisecond)
	third := time.After(3100 * time.Millisecond)
	fourth := time.After(4100 * time.Millisecond)

	// wait until all the waiting tasks run
	expectResponse := func(id string) {
		select {
		case got := <-runtimes:
			if got != id {
				t.Errorf("expected: %s got: %s", id, got)
			}
		default:
			t.Errorf("%s not ready", id)
		}
	}

	<-first
	expectResponse("a")

	<-second
	expectResponse("b")

	<-third
	expectResponse("c")
	expectResponse("d")

	<-fourth
	expectResponse("e")

	//close(runtimes)
	//expectResponse(0)
	// tell the scheduler we are all done
	s.Stop()
}
