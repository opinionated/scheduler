package scheduler_test

import (
	"time"
	"testing"
	"github.com/opinionated/scheduler/scheduler"
)
// Tests to make sure the Rater works properly
func TestRater(T *testing.T){
	// creates a map to test corner cases on
	mymap := make(map[string]time.Time)
	// Sets upper limit for time ahead allowed
	var timeleft time.Duration = 22 * time.Hour
	// Tests cases for Current time, day after, and day before
	mymap["NewyorkTimes.com"] = time.Now()
	mymap["Swagmoney.com"] = (time.Now()).AddDate(0,0,1)
	mymap["Cnn.com"] = (time.Now()).AddDate(0,0,-1)
	// Makes the struct
	var m scheduler.Rater = scheduler.Rater{mymap, timeleft}
	// Tests to make sure CanRun Works for these values
	if m.CanRun("NewyorkTimes.com"){
		T.Errorf("Day Before Failed\n")
	}
	if !(m.CanRun("Cnn.com")){
		T.Errorf("Right now Failed\n")
	}
	if (m.CanRun("Swagmoney.com")){
		T.Errorf("Tomorrow Failed\n")
	}
}