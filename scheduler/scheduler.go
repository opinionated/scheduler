package scheduler

import (
	"fmt"
	"sort"
	"time"
)

// Used with the Scheduler to delay running tasks so a server doesn't get bombarded
// Might want to control limits by website so its harder to accidentally spam a website
type Schedulable interface {
	DoWork(*Scheduler)     // runs the task
	GetTimeRemaining() int // seconds to run
	SetTimeRemaining(int)  // sets the number of seconds for the task to run
	IsLoopable() bool      // does the schedulable get removed once run
	// if something is loopable it has to reset the loop timer when it starts
}

// Implements sort.Sort
// Syntax for sort is By(func).Sort(array)
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

// Sorts time remaining low -> high
// Used by the scheduler
func SortLowToHigh(s1, s2 Schedulable) bool {
	return s1.GetTimeRemaining() < s2.GetTimeRemaining()
}

// Manages schedulable tasks ie tasks that you want to run some time in the future
// Scheduling burden is on the programmer, might want to create some tools to change that
// Timing isn't very tight
type Scheduler struct {
	queue     []Schedulable    // sorted array of tasks to run
	addTask   chan Schedulable // tasks are put on here when they are able to run
	quit      chan bool        // signal the scheduler to stop once no more tasks are ready
	ready     chan Schedulable // the next task to run. This needs to be buffered or deadlock will occur
	isRunning bool             // if the scheduler is running
}

// Create and return a scheduler
// NOTE: go figures out that you are returning a pointer to the local variable and puts it on the heap for you
// queue size is how many tasks can be held TODO: check if this is fixed or can expand
// bufferSize is how many tasks can be added to the queue each cycle.
// 	if bufferSize < num tasks being added at once then some of the adds will block unless run as goroutines
// 	choice of buffer size shouldn't make a huge difference
func MakeScheduler(queueSize, bufferSize int) *Scheduler {
	return &Scheduler{make([]Schedulable, 0, queueSize),
		make(chan Schedulable, bufferSize),
		make(chan bool),
		make(chan Schedulable, 1),
		false}
}

// threadsafe add, may block if addTask is buffered. In this case, run it asynchronously as a go routine
func (scheduler *Scheduler) AddSchedulable(schedulable Schedulable) {
	scheduler.addTask <- schedulable
}

// start running the scheduler asynchronously
func (scheduler *Scheduler) Start() {
	go scheduler.Run()
}

// stop the scheduler
func (scheduler *Scheduler) Stop() {
	scheduler.quit <- true
}

func (scheduler *Scheduler) IsRunning() bool {
	return scheduler.isRunning
}

// Body of the scheduler. Manage it with the start and stop functions
// TODO: change constants to reflect seconds, not ms (used ms for testing)
func (scheduler *Scheduler) Run() {
	scheduler.isRunning = true
	for {
		// add any new tasks
		didAdd := false // keep track of adds so we only sort when we need to
	AddNewTasksLoop:
		for {
			// add tasks from buffered channel to queue until all waiting tasks are added
			select {
			case s := <-scheduler.addTask:
				scheduler.queue = append(scheduler.queue, s)
				didAdd = true
			default:
				// break out of the for loop
				break AddNewTasksLoop
			}
		}

		// only sort if we added a new task
		if didAdd {
			By(SortLowToHigh).Sort(scheduler.queue)
		}

		// get the next task to run
		// TODO: change cycletime to seconds once done testing
		var cycleTime int = 1 // how often the scheduler loops while idle
		if len(scheduler.queue) > 0 {
			if scheduler.queue[0].GetTimeRemaining() < cycleTime {
				// if a task will be ready this cycle, add run it
				scheduler.ready <- scheduler.queue[0]

				// remove the first element
				// do it this way to make sure we avoid mem leaks
				// (something could be sitting in an unused part of the queue and not get cleared)
				copy(scheduler.queue[0:], scheduler.queue[1:])
				scheduler.queue[len(scheduler.queue)-1] = nil
				scheduler.queue = scheduler.queue[:len(scheduler.queue)-1]
			}
		}

		// Run any tasks that are waiting
		// stop running if we got signal to stop and no tasks are waiting
		// if no tasks got run wait one timestep (cycleTime) to not burn CPU
		select {
		case task := <-scheduler.ready:
			// assume task gets removed from queue when it is put into the channel
			go task.DoWork(scheduler)
		case <-scheduler.quit:
			scheduler.isRunning = false
			fmt.Println("Done with scheduler")
			return
		default:
			time.Sleep(time.Duration(cycleTime) * time.Second)
		}
	}
}
