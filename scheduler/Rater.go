package scheduler

import "time"
// Struct to check if a user is checking a website too frequently
type Rater struct {
	// Map of website names to last check times
	LastChecked map[string]time.Time
	// Largest amount of time allowed between each check
	AcceptableLimit time.Duration
}
// Checks if you can access the website based on the Acceptable Limit
func (rates Rater) CanRun(website string ) bool {
	return time.Since(rates.LastChecked[website]) >= rates.AcceptableLimit
}
