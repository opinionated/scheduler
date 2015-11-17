package scheduler_test

import (
	"time"
	"testing"
	"github.com/opinionated/scheduler/scheduler"
)

func TestRater(T *testing.T){
	mymap := make(map[string]time.Time)
	var timeleft time.Duration = 22 * time.Hour
	mymap["NewyorkTimes.com"] = time.Now()
	mymap["Swagmoney.com"] = (time.Now()).AddDate(0,0,1)
	mymap["Cnn.com"] = (time.Now()).AddDate(0,0,-1)
	var m scheduler.Rater = scheduler.Rater{mymap, timeleft}
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