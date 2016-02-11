package scheduler

import (
	"sort"
)

// Schedulable wraps tasks run by the Scheduler. It provides timing methods for
// one off and periodic tasks.
type Schedulable interface {
	// DoWork is run asynchronously by the scheduler when it is ready. Override this.
	Run(*Scheduler)
	// GetTimeRemaining before it is ready to run in seconds
	TimeRemaining() int
	// SetTimeRemaining number of seconds from now until the task should run.
	SetTimeRemaining(int)
}

// SchedulableSorter implements sort for Schedulable tasks.
type SchedulableSorter struct {
	queue []Schedulable
	by    func(s1, s2 Schedulable) bool
}

// By is used to sort.
type By func(s1, s2 Schedulable) bool

// Sort sorts an array of schedulables
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
	return s1.TimeRemaining() < s2.TimeRemaining()
}
