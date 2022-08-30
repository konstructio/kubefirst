package progressPrinter

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

// Struct used to manage tracker object
// This object may evolve with more properties in th future
// when we have more fancier UI tools/styles.
type ActionTracker struct {
	Tracker *progress.Tracker
}

// General state object
type progressPrinter struct {
	Trackers map[string]*ActionTracker
	pw       progress.Writer
	noTTY    bool
}

var instance *progressPrinter
var once sync.Once

// Function used to initialize the component once in the execution.
// Usually called from the `cmd`  `init` func or as early as possible on the execution.
//
//	import ("github.com/kubefirst/nebulous/pkg")
//	func init() {
//			progressPrinter.GetInstance()
//			progressPrinter.SetupProgress(5) // Number of bars for the entire run.
//	}
func GetInstance() *progressPrinter {
	once.Do(func() {
		instance = &progressPrinter{}
		instance.Trackers = make(map[string]*ActionTracker)
	})
	return instance
}

// SetupProgress prepare the progress bar setting its initial configuration
// Used for general initialization of tracker object and overall counter
func SetupProgress(numTrackers int, noTTY bool) {
	flag.Parse()
	fmt.Printf("Init actions: %d expected tasks ...\n\n", numTrackers)
	// instantiate a Progress Writer and set up the options
	instance.pw = progress.NewWriter()

	// if silent mode, dont show progress bar render
	if noTTY {
		instance.noTTY = true
		return
	} else {
		instance.noTTY = false
	}

	instance.pw.SetAutoStop(false)
	instance.pw.SetTrackerLength(30)
	instance.pw.SetMessageWidth(29)
	instance.pw.SetNumTrackersExpected(numTrackers)
	instance.pw.SetSortBy(progress.SortByPercentDsc)
	instance.pw.SetStyle(progress.StyleDefault)
	instance.pw.SetTrackerPosition(progress.PositionRight)
	instance.pw.SetUpdateFrequency(time.Millisecond * 100)
	instance.pw.Style().Colors = progress.StyleColorsExample
	instance.pw.Style().Options.PercentFormat = "%4.1f%%"
	instance.pw.Style().Visibility.ETA = false
	instance.pw.Style().Visibility.ETAOverall = false
	instance.pw.Style().Visibility.Percentage = true
	instance.pw.Style().Visibility.Time = true
	instance.pw.Style().Visibility.TrackerOverall = true
	instance.pw.Style().Visibility.Value = true
	go instance.pw.Render()
}

//	Initialise a tracker object
//
// Prefer `AddTracker` to create trackers, due to simplicity.
func CreateTracker(title string, total int64) *progress.Tracker {
	tracker := &progress.Tracker{
		Message: title,
		Total:   total,
		Units:   progress.UnitsDefault,
	}

	instance.pw.AppendTracker(tracker)
	return tracker
}

// Prints a log message near the current active tracker.
// Sample of usage:
//
//	progressPrinter.LogMessage("- Waiting bootstrap")
func LogMessage(message string) {
	if !instance.noTTY {
		instance.pw.Log(message)
	} else {
		fmt.Println(message)
	}

}

// Add Tracker (prefered way)
// Return a string for the key to be used on future uses
// Sample of usage:
//
//	progressPrinter.AddTracker("step-base", "Apply Base ", 3)
//
// no need to instanciate, it is a singleton, only one instance already started before use.
func AddTracker(key string, title string, total int64) string {
	time.Sleep(1 * time.Second)
	instance.Trackers[key] = &ActionTracker{Tracker: CreateTracker(title, total)}
	return key
}

// Increments a tracker based on the provided key
// if key is unkown it will error out.
// Sample of usage:
//
//	progressPrinter.IncrementTracker("step-base", 1)
func IncrementTracker(key string, value int64) {
	time.Sleep(1 * time.Second)
	instance.Trackers[key].Tracker.Increment(int64(value))
	if instance.noTTY {
		fmt.Printf("- %s [incremented: %d]\n", instance.Trackers[key].Tracker.Message, value)
	}
}
