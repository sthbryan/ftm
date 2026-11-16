// Package registry lists the tunnel providers ftm can run, one adapter per
// core.Provider.
package registry

import (
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/provider/bore"
	"github.com/sthbryan/ftm/internal/provider/cloudflared"
	"github.com/sthbryan/ftm/internal/provider/pinggy"
	"github.com/sthbryan/ftm/internal/provider/ssh"
	"github.com/sthbryan/ftm/internal/provider/tunnelmole"
)

// Default returns a fresh adapter for every provider in core.AllProviders.
func Default() map[core.Provider]provider.Provider {
	return map[core.Provider]provider.Provider{
		core.ProviderCloudflared:  cloudflared.New(),
		core.ProviderTunnelmole:   tunnelmole.New(),
		core.ProviderLocalhostRun: ssh.NewLocalhostRun(),
		core.ProviderServeo:       ssh.NewServeo(),
		core.ProviderPinggy:       pinggy.New(),
		core.ProviderBore:         bore.New(),
	}
}
