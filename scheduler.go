// Package scheduler provides a periodic task and implements a task scheduler. Its
// primary purpose is to manage the volume of scraping requests to a target.
package scheduler

import (
	"github.com/opinionated/utils/log"
	"time"
)

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

// Add adds a new schedulable to the run queue. May block if the waiting queue is full.
func (scheduler *Scheduler) Add(schedulable Schedulable) {
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

		i := 0 // declare outside so we can use later
		didRemove := false
		for ; i < len(scheduler.queue); i++ {
			if scheduler.queue[i].TimeRemaining() < scheduler.cycleTime {
				// run any tasks that are ready
				go scheduler.queue[i].Run(scheduler)
				didRemove = true

			} else { // no tasks to run this cycle
				break
			}

		}

		if didRemove {
			// remove unused elements
			// do it this way to make sure we avoid mem leaks
			// (something could be sitting in an unused part of the queue and not get cleared)
			copy(scheduler.queue[0:], scheduler.queue[i:])
			for j := 1; j <= i; j++ {
				scheduler.queue[len(scheduler.queue)-j] = nil
			}
			scheduler.queue = scheduler.queue[:len(scheduler.queue)-i]
		}

		select {
		case <-ticker.C:
			// wait until next time step

		case <-scheduler.quit:
			scheduler.isRunning = false
			log.Warn("Done with scheduler")
			ticker.Stop()
			return
		}
	}
}
