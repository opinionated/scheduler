package scheduler

import "time"

type Rater struct {
	LastChecked map[string]time.Time
	AcceptableLimit time.Duration
}

func (rates Rater) CanRun(website string ) bool {
	return time.Since(rates.LastChecked[website]) >= rates.AcceptableLimit
}
