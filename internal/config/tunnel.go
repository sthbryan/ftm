package config

type Provider string

const (
	ProviderPlayitgg    Provider = "playitgg"
	ProviderCloudflared Provider = "cloudflared"
	ProviderTunnelmole  Provider = "tunnelmole"
	ProviderLocalhostRun Provider = "localhostrun"
	ProviderServeo      Provider = "serveo"
	ProviderPinggy      Provider = "pinggy"
)

type TunnelConfig struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Provider   Provider `yaml:"provider"`
	LocalPort  int      `yaml:"local_port"`
	ShortURL   string   `yaml:"short_url,omitempty"`
	AutoStart  bool     `yaml:"auto_start"`
	CustomArgs []string `yaml:"custom_args,omitempty"`
}

type TunnelStatus struct {
	ID           string
	Name         string
	Provider     Provider
	LocalPort    int
	PublicURL    string
	ShortURL     string
	Running      bool
	Starting     bool
	Stopping     bool
	Error        string
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
		ShortURL:   tc.ShortURL,
		MaxPlayers: 8,
	}
}
