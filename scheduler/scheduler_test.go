package scheduler

import "testing"
import "time"
import "fmt"

// Type used for testing the scheduler
type TestSchedulable struct {
	start time.Time
	delay float64
	msg   string
}

// simple factory function
func MakeTestSchedulable(delay int, id string) TestSchedulable {
	return TestSchedulable{time.Now(), float64(delay), id}
}

// double check we implement all the interfaces
var _ Schedulable = (*TestSchedulable)(nil)

// check if we are ready to run
func (schedulable TestSchedulable) GetTimeRemaining() int {
	// convert this to ms
	del := schedulable.delay - time.Since(schedulable.start).Seconds()*1000
	if del < 0 {
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
func (schedulable TestSchedulable) DoWork(scheduler *Scheduler) {
	fmt.Println("doing task:", schedulable.msg)
}

// check that the scheduler is threadsafe, runs tasks in the right order
func TestScheduler(t *testing.T) {
	//s := &Scheduler{make([]Schedulable, 0, 5), make(chan Schedulable, 3), make(chan bool), make(chan Schedulable, 1)}

	s := MakeScheduler(5, 3)
	s.Start()
	go s.AddSchedulable(MakeTestSchedulable(4000, "d"))
	go s.AddSchedulable(MakeTestSchedulable(2200, "c"))
	go s.AddSchedulable(MakeTestSchedulable(2000, "a"))
	go s.AddSchedulable(MakeTestSchedulable(2100, "b"))

	// wait until all the waiting tasks run
	time.Sleep(5 * time.Second)

	// tell the scheduler we are all done
	s.Stop()
}

//asdf
