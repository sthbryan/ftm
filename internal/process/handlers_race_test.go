package process

import (
	"sync"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

// The manager is reconfigured from the UI while the process watchers are
// notifying, so setters must be safe against concurrent readers. Run with
// -race to make this meaningful.
func TestHandlersCanBeReplacedWhileNotifying(t *testing.T) {
	m := NewManager()
	ch := make(chan config.TunnelStatus, 1)
	m.SetStatusChannel(ch)
	m.SetProgressChannel(make(chan providers.DownloadProgress, 1))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.SetNotificationHandler(func(config.TunnelStatus) {})
			m.SetVisitorHandler(func(config.TunnelStatus) {})
			m.SetStatusChannel(ch)
			m.SetExpirationCallbacks(
				func(string, string, string, time.Time) {},
				func(string) {},
			)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		status := config.TunnelStatus{ID: "t1", State: config.TunnelStateOnline}
		for i := 0; i < 200; i++ {
			m.callNotificationHandler(status)
			m.callStatusUpdate(status)
			m.callExpirationStart("t1", "Foundry", "cloudflared", time.Now())
			m.callExpirationStop("t1")
			<-ch
		}
	}()

	wg.Wait()
}
