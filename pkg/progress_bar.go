package pkg

import (
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/progress"
	"time"
)

type ActionTracker struct {
	Tracker *progress.Tracker
}

const TrackerStage0 = "1 - Load properties"
const TrackerStage1 = "2 - Set .flare initial values"
const TrackerStage2 = "3 - Test Domain Liveness"
const TrackerStage3 = "4 - Create SSH Key Pair"
const TrackerStage4 = "5 - Load Templates"
const TrackerStage5 = "6 - DownloadTools Tools"
const TrackerStage6 = "7 - Get Account Info"
const TrackerStage7 = "8 - Create Buckets"
const TrackerStage8 = "9 - Detokenize"
const TrackerStage9 = "10 - Send Telemetry"

//const trackerStage5 = "6 - DownloadTools Tools"

const TrackerStage20 = "0 - Apply Base"

//const trackerStage21 = "1 - Temporary SCM Install"
//const trackerStage22 = "2 - Argo/Final SCM Install"
//const trackerStage23 = "3 - Final Setup"

var (
	pw                     progress.Writer
	Trackers               map[string]*ActionTracker
	flagAutoStop           = flag.Bool("auto-stop", false, "Auto-stop rendering?")
	flagHideETA            = flag.Bool("hide-eta", false, "Hide the ETA?")
	flagHideETAOverall     = flag.Bool("hide-eta-overall", false, "Hide the ETA in the overall tracker?")
	flagHideOverallTracker = flag.Bool("hide-overall", false, "Hide the Overall Tracker?")
	flagHidePercentage     = flag.Bool("hide-percentage", false, "Hide the progress percent?")
	flagHideTime           = flag.Bool("hide-time", false, "Hide the time taken?")
	flagHideValue          = flag.Bool("hide-value", false, "Hide the tracker value?")
	//	flagNumTrackers        = flag.Int("num-trackers", 12, "Number of Trackers")
	flagRandomFail = flag.Bool("rnd-fail", false, "Introduce random failures in tracking")
	flagRandomLogs = flag.Bool("rnd-logs", false, "Output random logs in the middle of tracking")
)

// GetTrackers keeps one single instance of Trackers alive using singleton pattern.
func GetTrackers() map[string]*ActionTracker {
	if Trackers != nil {
		return Trackers
	}
	Trackers = make(map[string]*ActionTracker)
	return Trackers
}

// CreateTracker receives Tracker data and return a new Tracker.
func CreateTracker(title string, total int64) *progress.Tracker {

	tracker := &progress.Tracker{
		Message: title,
		Total:   total,
		Units:   progress.UnitsDefault,
	}

	pw.AppendTracker(tracker)

	return tracker
}

// SetupProgress prepare the progress bar setting its initial configuration
func SetupProgress(numTrackers int) {

	flagNumTrackers := flag.Int("num-trackers", numTrackers, "Number of Trackers")
	flag.Parse()
	fmt.Printf("Init actions: %d expected tasks ...\n\n", *flagNumTrackers)
	// instantiate a Progress Writer and set up the options
	pw = progress.NewWriter()
	pw.SetAutoStop(*flagAutoStop)
	pw.SetTrackerLength(30)
	pw.SetMessageWidth(29)
	pw.SetNumTrackersExpected(*flagNumTrackers)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Visibility.ETA = !*flagHideETA
	pw.Style().Visibility.ETAOverall = !*flagHideETAOverall
	pw.Style().Visibility.Percentage = !*flagHidePercentage
	pw.Style().Visibility.Time = !*flagHideTime
	pw.Style().Visibility.TrackerOverall = !*flagHideOverallTracker
	pw.Style().Visibility.Value = !*flagHideValue
	go pw.Render()
}
