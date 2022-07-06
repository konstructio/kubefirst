package flare

import (
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
	"time"
)

type ActionTracker struct {
	Tracker *progress.Tracker
}

var (
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

	messageColors = []text.Color{
		text.FgRed,
		text.FgGreen,
		text.FgYellow,
		text.FgBlue,
		text.FgMagenta,
		text.FgCyan,
		text.FgWhite,
	}
)
var pw progress.Writer

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

func CreateTracker(title string, total int64) *progress.Tracker {
	units := &progress.UnitsDefault
	message := title
	tracker := progress.Tracker{Message: message, Total: total, Units: *units}
	pw.AppendTracker(&tracker)
	return &tracker
}
