package config

type Provider string

const (
	ProviderPlayitgg     Provider = "playitgg"
	ProviderCloudflared  Provider = "cloudflared"
	ProviderTunnelmole   Provider = "tunnelmole"
	ProviderLocalhostRun Provider = "localhostrun"
	ProviderServeo       Provider = "serveo"
	ProviderPinggy       Provider = "pinggy"
)

type TunnelState string

const (
	TunnelStateNone       TunnelState = ""
	TunnelStateDownload   TunnelState = "downloading"
	TunnelStateInstall    TunnelState = "installing"
	TunnelStateStarting   TunnelState = "starting"
	TunnelStateConnecting TunnelState = "connecting"
	TunnelStateOnline     TunnelState = "online"
	TunnelStateStopping   TunnelState = "stopping"
	TunnelStateStopped    TunnelState = "stopped"
	TunnelStateTimeout    TunnelState = "timeout"
	TunnelStateError      TunnelState = "error"
)

type TunnelConfig struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Provider   Provider `yaml:"provider"`
	LocalPort  int      `yaml:"local_port"`
	AutoStart  bool     `yaml:"auto_start"`
	CustomArgs []string `yaml:"custom_args,omitempty"`
	Order      int      `yaml:"order,omitempty"`
}

type TunnelStatus struct {
	ID           string
	Name         string
	Provider     Provider
	LocalPort    int
	PublicURL    string
	State        TunnelState
	ErrorMessage string
	LogLines     []string
	Players      int
	MaxPlayers   int
}

func (tc *TunnelConfig) Status() TunnelStatus {
	return TunnelStatus{
		ID:         tc.ID,
		Name:       tc.Name,
		Provider:   tc.Provider,
		LocalPort:  tc.LocalPort,
		MaxPlayers: 8,
		State:      TunnelStateStopped,
	}
}
