package web

import (
	"encoding/json"
	"log"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

// tunnelStateMessage is the only shape a tunnel state takes on the wire. The
// REST list, the WebSocket broadcast and the update path all build it here, so
// the payload cannot drift between them or from the TypeScript types.
type tunnelStateMessage struct {
	Type           string `json:"type"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	Port           int    `json:"port"`
	State          string `json:"state"`
	PublicURL      string `json:"publicUrl"`
	ErrorMessage   string `json:"errorMessage"`
	ExpiresAt      int64  `json:"expiresAt"`
	StartedAt      int64  `json:"startedAt"`
	Visitors       int    `json:"visitors,omitempty"`
	ActiveSessions int    `json:"activeSessions,omitempty"`
}

// tunnelStatePatch is for the short-lived states that are not driven by the
// stored tunnel, such as the installing and install-failure updates.
type tunnelStatePatch struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	State        string `json:"state"`
	Provider     string `json:"provider,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// tunnelStateFor merges a stored tunnel with its live status.
func tunnelStateFor(manager *process.Manager, t config.TunnelConfig) tunnelStateMessage {
	msg := tunnelStateMessage{
		Type:     "tunnel_state",
		ID:       t.ID,
		Name:     t.Name,
		Provider: string(t.Provider),
		Port:     t.LocalPort,
		State:    string(config.TunnelStateStopped),
	}

	if status, ok := manager.GetStatus(t.ID); ok {
		msg.State = string(status.State)
		msg.PublicURL = status.PublicURL
		msg.ErrorMessage = status.ErrorMessage
		msg.ExpiresAt = status.ExpiresAt
		msg.StartedAt = status.StartedAt
		msg.Visitors = status.Visitors
		msg.ActiveSessions = status.ActiveSessions
	}

	if msg.State == string(config.TunnelStateStopped) {
		if needsInstall, canInstall := manager.CheckInstallation(t.Provider); needsInstall && canInstall {
			msg.State = "need_installing"
		}
	}

	return msg
}

// tunnelStateFromStatus builds the message from a status the manager already
// emitted, which is what the status channel delivers.
func tunnelStateFromStatus(status config.TunnelStatus) tunnelStateMessage {
	return tunnelStateMessage{
		Type:         "tunnel_state",
		ID:           status.ID,
		Name:         status.Name,
		Provider:     string(status.Provider),
		Port:         status.LocalPort,
		State:        string(status.State),
		PublicURL:    status.PublicURL,
		ErrorMessage: status.ErrorMessage,
		ExpiresAt:    status.ExpiresAt,
		StartedAt:    status.StartedAt,
	}
}

// broadcastJSON encodes one update and drops it with a log line if it cannot be
// encoded, instead of silently discarding the error.
func (s *Server) broadcastJSON(v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		log.Printf("web: could not encode a %T update: %v", v, err)
		return
	}
	s.broadcast(string(payload))
}
