package stats

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// Reporter saves the counters and writes them to the log every interval, but
// only when something changed, so an idle server stays quiet.
type Reporter struct {
	stats    *Stats
	path     string
	interval time.Duration

	stop chan struct{}
	done chan struct{}
	once sync.Once

	lastLogged Snapshot
	logged     bool
}

// StartReporter begins reporting in the background. Call Stop to write the
// final numbers.
func StartReporter(s *Stats, path string, interval time.Duration) *Reporter {
	r := &Reporter{
		stats:    s,
		path:     path,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *Reporter) run() {
	defer close(r.done)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.report(false)
		case <-r.stop:
			r.report(true)
			return
		}
	}
}

func (r *Reporter) report(final bool) {
	snap := r.stats.Snapshot()
	if err := r.stats.Save(r.path); err != nil {
		log.Error("could not save usage stats", "error", err)
	}
	if !final && r.logged && snap == r.lastLogged {
		return
	}
	r.lastLogged, r.logged = snap, true
	log.Info("usage",
		"messages", snap.Messages,
		"logins", snap.Logins,
		"rejected", snap.Rejected,
		"refused", snap.Refused,
		"online", snap.Online,
	)
}

// Stop writes the final numbers and waits for the reporter to finish.
func (r *Reporter) Stop() {
	r.once.Do(func() { close(r.stop) })
	<-r.done
}
