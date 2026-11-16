package registry

import (
	"testing"

	"github.com/sthbryan/ftm/internal/core"
)

func TestDefaultCoversEveryProvider(t *testing.T) {
	adapters := Default()
	for _, id := range core.AllProviders() {
		if adapters[id] == nil {
			t.Errorf("no adapter for %s", id)
		}
	}
	if len(adapters) != len(core.AllProviders()) {
		t.Errorf("Default() has %d adapters, core.AllProviders() has %d", len(adapters), len(core.AllProviders()))
	}
}
