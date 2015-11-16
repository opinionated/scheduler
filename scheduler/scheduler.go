// Package scheduler provides a periodic task and implements a task scheduler. Its
// primary purpose is to manage the volume of scraping requests to a target.
package scheduler

import (
	"fmt"
	"sort"
	"time"
)

// Schedulable wraps tasks run by the Scheduler. It provides timing methods for
// one off and periodic tasks.
type Schedulable interface {
	// DoWork is run asynchronously by the scheduler when it is ready. Override this.
	DoWork(*Scheduler)
	// GetTimeRemaining before it is ready to run in seconds
	GetTimeRemaining() int
	// SetTimeRemaining number of seconds from now until the task should run.
	SetTimeRemaining(int)
}

// SchedulableSorter implements sort for Schedulable tasks.
type SchedulableSorter struct {
	queue []Schedulable
	by    func(s1, s2 Schedulable) bool
}

type By func(s1, s2 Schedulable) bool

func (by By) Sort(schedulables []Schedulable) {
	ss := &SchedulableSorter{
		queue: schedulables,
		by:    by,
	}
	sort.Sort(ss)
}

func (s *SchedulableSorter) Swap(i, j int) {
	s.queue[j], s.queue[i] = s.queue[i], s.queue[j]
}

func (s *SchedulableSorter) Len() int {
	return len(s.queue)
}

func (s *SchedulableSorter) Less(i, j int) bool {
	return s.by(s.queue[i], s.queue[j])
}

// SortLowToHigh sorts Schedulable tasks from shortest time remaining to longest
func SortLowToHigh(s1, s2 Schedulable) bool {
	return s1.GetTimeRemaining() < s2.GetTimeRemaining()
}

// Scheduler runs Schedulable tasks asynchronously at a specified time. Allows for dynamic
// task rescheduling and multithreaded task adding.
type Scheduler struct {
	queue     []Schedulable    // sorted array of tasks to run
	addTask   chan Schedulable // tasks are put on here when they are able to run
	quit      chan bool        // signal the scheduler to stop once no more tasks are ready
	ready     chan Schedulable // the next task to run. This needs to be buffered or deadlock will occur
	isRunning bool             // if the scheduler is running
	cycleTime int
}

// MakeScheduler creates a new Scheduler.
// QueueSize is how many tasks can be held.
// BufferSize is how many tasks can be added to the queue each cycle. If bufferSize < num tasks
// being added at once then some of the adds will block unless run as goroutines choice of buffer
// size shouldn't make a huge difference
func MakeScheduler(queueSize, bufferSize int) *Scheduler {
	// NOTE: go figures out that you are returning a pointer to the local variable and puts it on the heap for you
	return &Scheduler{make([]Schedulable, 0, queueSize),
		make(chan Schedulable, bufferSize),
		make(chan bool),
		make(chan Schedulable, 1),
		false,
		15}
}

// AddSchedulable adds a new schedulable to the run queue. May block if the waiting queue is full.
func (scheduler *Scheduler) AddSchedulable(schedulable Schedulable) {
	scheduler.addTask <- schedulable
}

// SetCycleTime of the main loop.
func (scheduler *Scheduler) SetCycleTime(time int) {
	scheduler.cycleTime = time
}

// Start the Scheduler asynchronously. Does not block.
func (scheduler *Scheduler) Start() {
	go scheduler.Run()
}

// Stop the scheduler from running. May block until Scheduler ends.
func (scheduler *Scheduler) Stop() {
	scheduler.quit <- true
}

// IsRunning returns true if the scheduler is running, otherwise returns flase.
func (scheduler *Scheduler) IsRunning() bool {
	return scheduler.isRunning
}

// Run the main scheduler loop.
func (scheduler *Scheduler) Run() {
	scheduler.isRunning = true

	// signals the loop to run every (cycleTime) seconds

	ticker := time.NewTicker(time.Duration(scheduler.cycleTime) * time.Second)

	for {

		didAdd := false // keep track of adds so we only sort when we need to
	AddNewTasksLoop:
		for {
			// add tasks from buffered channel to queue until all waiting tasks are added
			select {
			case s := <-scheduler.addTask:
				scheduler.queue = append(scheduler.queue, s)
				didAdd = true
			default:
				break AddNewTasksLoop
			}
		}

		// only sort if we added a new task
		// TODO: reschedule tasks properly
		if didAdd {
			By(SortLowToHigh).Sort(scheduler.queue)
		}

		for len(scheduler.queue) > 0 {
			if scheduler.queue[0].GetTimeRemaining() < scheduler.cycleTime {
				// run any tasks that are ready
				go scheduler.queue[0].DoWork(scheduler)

				// remove the first element
				// do it this way to make sure we avoid mem leaks
				// (something could be sitting in an unused part of the queue and not get cleared)
				copy(scheduler.queue[0:], scheduler.queue[1:])
				scheduler.queue[len(scheduler.queue)-1] = nil
				scheduler.queue = scheduler.queue[:len(scheduler.queue)-1]
			} else { // no tasks to run this cycle
				break
			}

		}

		select {
		case <-ticker.C:
			// wait until next time step

		case <-scheduler.quit:
			scheduler.isRunning = false
			fmt.Println("Done with scheduler")
			ticker.Stop()
			return
		}
	}
}
