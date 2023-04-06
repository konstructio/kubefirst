/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progressPrinter

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rs/zerolog/log"
)

// ActionTracker Struct used to manage tracker object
// This object may evolve with more properties in th future
// when we have more fancier UI tools/styles.
type ActionTracker struct {
	Tracker *progress.Tracker
}

// progressPrinter General state object
type progressPrinter struct {
	Trackers map[string]*ActionTracker
	pw       progress.Writer
}

var instance *progressPrinter
var once sync.Once

// GetInstance  Function used to initialize the component once in the execution.
// Usually called from the `cmd`  `init` func or as early as possible on the execution.
//
//	import ("github.com/kubefirst/kubefirst/pkg")
//	func init() {
//			progressPrinter.GetInstance()
//			progressPrinter.SetupProgress(5) // Number of bars for the entire run.
//	}
func GetInstance() *progressPrinter {
	once.Do(func() {
		instance = &progressPrinter{}
		instance.Trackers = make(map[string]*ActionTracker)
		// instantiate a Progress Writer and set up the options
		instance.pw = progress.NewWriter()

	})
	return instance
}

// SetupProgress prepare the progress bar setting its initial configuration
// Used for general initialization of tracker object and overall counter
func SetupProgress(numTrackers int, silentMode bool) {
	flag.Parse()
	log.Debug().Msg(fmt.Sprintf("Init actions: %d expected tasks ...\n\n", numTrackers))
	// if silent mode, dont show progress bar render
	if silentMode {
		return
	}

	instance.pw.SetAutoStop(false)
	instance.pw.SetTrackerLength(40)
	instance.pw.SetMessageWidth(39)
	instance.pw.SetNumTrackersExpected(numTrackers)
	instance.pw.SetSortBy(progress.SortByPercentDsc)
	instance.pw.SetStyle(progress.StyleDefault)
	instance.pw.SetTrackerPosition(progress.PositionRight)
	instance.pw.SetUpdateFrequency(time.Millisecond * 100)
	instance.pw.Style().Colors = progress.StyleColors{
		Message: text.Colors{text.FgWhite},
		Error:   text.Colors{text.FgRed},
		Percent: text.Colors{text.FgCyan},
		Stats:   text.Colors{text.FgHiBlack},
		Time:    text.Colors{text.FgGreen},
		Tracker: text.Colors{text.FgYellow},
		Value:   text.Colors{text.FgCyan},
	}
	instance.pw.Style().Options.PercentFormat = "%4.1f%%"
	instance.pw.Style().Visibility.ETA = false
	instance.pw.Style().Visibility.ETAOverall = false
	instance.pw.Style().Visibility.Percentage = true
	instance.pw.Style().Visibility.Time = true
	instance.pw.Style().Visibility.TrackerOverall = true
	instance.pw.Style().Visibility.Value = true
	go instance.pw.Render()
}

// CreateTracker Initialise a tracker object
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

// LogMessage Prints a log message near the current active tracker.
// Sample of usage:
//
//	progressPrinter.LogMessage("- Waiting bootstrap")
func LogMessage(message string) {
	instance.pw.Log(message)
}

// AddTracker Add Tracker (prefered way)
// Return a string for the key to be used on future uses
// Sample of usage:
//
//	progressPrinter.AddTracker("step-base", "Apply Base ", 3)
//
// no need to instanciate, it is a singleton, only one instance already started before use.
func AddTracker(key string, title string, total int64) string {
	instance.Trackers[key] = &ActionTracker{Tracker: CreateTracker(title, total)}
	return key
}

// TotalOfTrackers Returns the number of initialized Trackers
func TotalOfTrackers() int {
	return len(instance.Trackers)
}

// IncrementTracker Increments a tracker based on the provided key
// if key is unkown it will error out.
// Sample of usage:
//
//	progressPrinter.IncrementTracker("step-base", 1)
func IncrementTracker(key string, value int64) {
	instance.Trackers[key].Tracker.Increment(int64(1))
}
