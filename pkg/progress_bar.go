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

const DownloadDependencies = "Download dependencies"
const GetAccountInfo = "Get Account Info"
const GetDNSInfo = "Get DNS Info"
const TestHostedZoneLiveness = "Test Domain Liveness"
const CloneAndDetokenizeGitOpsTemplate = "Clone and Detokenize (GitOps)"
const CloneAndDetokenizeMetaphorTemplate = "Clone and Detokenize (Metaphor)"
const CreateSSHKey = "Create SSH keys"
const CreateBuckets = "Create Buckets"
const Detokenization = "Detokenization"
const SendTelemetry = "Send Telemetry"

const TrackerStage20 = "Apply Base"

//const GetAccountInfo = "1 - Set .flare initial values"
//const CreateSSHKey = "1 - Load properties"
//const TestHostedZoneLiveness = "4 - Create SSH Key Pair"
//const TrackerStage4 = "5 - Load Templates"
//const CloneAndDetokenizeMetaphorTemplate = "8 - Create Buckets"
//const Detokenize1 = "9 - Detokenize"
//const trackerStage5 = "6 - DownloadTools Tools"

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
	pw.SetTrackerLength(40)
	pw.SetMessageWidth(39)
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
