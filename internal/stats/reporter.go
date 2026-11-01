package stats

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// DefaultSaveInterval is how often the totals are written to disk. It is much
// shorter than the logging schedule so a crash loses minutes of counts, not a
// day.
const DefaultSaveInterval = 5 * time.Minute

// Reporter saves the counters to disk regularly and writes them to the log
// once a day at midnight UTC. It also saves and logs once more on Stop, so a
// redeploy leaves the final numbers in the old container's logs.
type Reporter struct {
	stats        *Stats
	path         string
	saveInterval time.Duration
	untilLog     func() time.Duration
	logf         func(Snapshot)

	stop chan struct{}
	done chan struct{}
	once sync.Once
}

// StartReporter begins reporting in the background. Call Stop to write the
// final numbers.
func StartReporter(s *Stats, path string) *Reporter {
	return startReporter(s, path, DefaultSaveInterval, untilNextMidnightUTC, logSnapshot)
}

func startReporter(s *Stats, path string, saveInterval time.Duration, untilLog func() time.Duration, logf func(Snapshot)) *Reporter {
	r := &Reporter{
		stats:        s,
		path:         path,
		saveInterval: saveInterval,
		untilLog:     untilLog,
		logf:         logf,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *Reporter) run() {
	defer close(r.done)
	save := time.NewTicker(r.saveInterval)
	defer save.Stop()
	for {
		logTimer := time.NewTimer(r.untilLog())
		for waiting := true; waiting; {
			select {
			case <-save.C:
				r.save()
			case <-logTimer.C:
				r.save()
				r.logf(r.stats.Snapshot())
				waiting = false
			case <-r.stop:
				logTimer.Stop()
				r.save()
				r.logf(r.stats.Snapshot())
				return
			}
		}
	}
}

func (r *Reporter) save() {
	if err := r.stats.Save(r.path); err != nil {
		log.Error("could not save usage stats", "error", err)
	}
}

// Stop writes the final numbers and waits for the reporter to finish.
func (r *Reporter) Stop() {
	r.once.Do(func() { close(r.stop) })
	<-r.done
}

func logSnapshot(snap Snapshot) {
	log.Info("usage",
		"messages", snap.Messages,
		"logins", snap.Logins,
		"rejected", snap.Rejected,
		"refused", snap.Refused,
		"online", snap.Online,
	)
}

// untilNextMidnightUTC is the time until the next 00:00 UTC.
func untilNextMidnightUTC() time.Duration {
	return untilMidnightUTC(time.Now())
}

func untilMidnightUTC(now time.Time) time.Duration {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return next.Sub(now)
}
