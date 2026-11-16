package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/service"
)

const defaultFoundryPort = 30000

type tunnelIDInput struct {
	ID string `path:"id" doc:"Tunnel id"`
}

type tunnelBody struct {
	_         struct{} `json:"-" additionalProperties:"true"`
	Name      string   `json:"name" required:"false"`
	Provider  string   `json:"provider" required:"false" doc:"cloudflared, tunnelmole, localhostrun, serveo, pinggy or bore"`
	LocalPort int      `json:"localPort" required:"false" doc:"Port Foundry listens on"`
}

type createTunnelInput struct {
	Body tunnelBody
}

type updateTunnelInput struct {
	ID   string `path:"id" doc:"Tunnel id"`
	Body tunnelBody
}

type tunnelOutput struct {
	Body TunnelState
}

type tunnelListOutput struct {
	Body []TunnelState
}

type clearTunnelsOutput struct {
	Body struct {
		Deleted int `json:"deleted" doc:"How many connections were removed"`
	}
}

type emptyOutput struct{}

type optionalTunnelOutput struct {
	Body *TunnelState
}

func errTunnelNotFound() *Error {
	return apiError(http.StatusNotFound, "tunnel_not_found", "tunnel not found")
}

func errSaveFailed(err error) *Error {
	log.Printf("httpapi: could not save the config: %v", err)
	return apiError(http.StatusInternalServerError, "save_failed", "Failed to save config")
}

func (s *Server) tunnelState(id string) (TunnelState, bool) {
	view, ok := s.svc.TunnelView(id)
	if !ok {
		return TunnelState{}, false
	}
	return tunnelStateFromView(view), true
}

func (s *Server) registerTunnels(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-tunnels",
		Method:      http.MethodGet,
		Path:        "/api/tunnels",
		Summary:     "List saved connections with their live state",
		Tags:        []string{tagTunnels},
	}, func(ctx context.Context, _ *struct{}) (*tunnelListOutput, error) {
		views := s.svc.TunnelViews()
		out := &tunnelListOutput{Body: make([]TunnelState, 0, len(views))}
		for _, v := range views {
			out.Body = append(out.Body, tunnelStateFromView(v))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-tunnel",
		Method:      http.MethodPost,
		Path:        "/api/tunnels",
		Summary:     "Save a new connection",
		Description: "name and provider are required; localPort defaults to 30000.",
		Tags:        []string{tagTunnels},
	}, func(ctx context.Context, in *createTunnelInput) (*tunnelOutput, error) {
		b := in.Body
		if strings.TrimSpace(b.Name) == "" || b.Provider == "" {
			return nil, apiError(http.StatusBadRequest, "missing_fields", "missing fields")
		}
		if b.LocalPort == 0 {
			b.LocalPort = defaultFoundryPort
		}
		if b.LocalPort < 1 || b.LocalPort > 65535 {
			return nil, apiError(http.StatusBadRequest, "invalid_port", "invalid port")
		}

		tunnel, err := s.svc.CreateTunnel(service.TunnelInput{
			Name:      b.Name,
			Provider:  core.Provider(b.Provider),
			LocalPort: b.LocalPort,
		})
		if err != nil {
			return nil, errSaveFailed(err)
		}
		return &tunnelOutput{Body: tunnelStateFromView(s.svc.View(tunnel))}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "clear-tunnels",
		Method:      http.MethodDelete,
		Path:        "/api/tunnels",
		Summary:     "Stop and remove every saved connection",
		Tags:        []string{tagTunnels},
	}, func(ctx context.Context, _ *struct{}) (*clearTunnelsOutput, error) {
		deleted, err := s.svc.ClearTunnels()
		if err != nil {
			return nil, errSaveFailed(err)
		}
		out := &clearTunnelsOutput{}
		out.Body.Deleted = deleted
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-tunnel",
		Method:      http.MethodGet,
		Path:        "/api/tunnels/{id}",
		Summary:     "Get one connection with its live state",
		Tags:        []string{tagTunnels},
	}, func(ctx context.Context, in *tunnelIDInput) (*tunnelOutput, error) {
		state, ok := s.tunnelState(in.ID)
		if !ok {
			return nil, errTunnelNotFound()
		}
		return &tunnelOutput{Body: state}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-tunnel",
		Method:      http.MethodPut,
		Path:        "/api/tunnels/{id}",
		Summary:     "Edit a saved connection",
		Description: "Empty fields and an out-of-range port keep the saved value. A running tunnel " +
			"keeps its old settings until it is restarted.",
		Tags: []string{tagTunnels},
	}, func(ctx context.Context, in *updateTunnelInput) (*tunnelOutput, error) {
		var patch service.TunnelPatch
		if name := in.Body.Name; name != "" {
			patch.Name = &name
		}
		if in.Body.Provider != "" {
			provider := core.Provider(in.Body.Provider)
			patch.Provider = &provider
		}
		if port := in.Body.LocalPort; port >= 1 && port <= 65535 {
			patch.LocalPort = &port
		}

		tunnel, err := s.svc.UpdateTunnel(in.ID, patch)
		if errors.Is(err, service.ErrTunnelNotFound) {
			return nil, errTunnelNotFound()
		}
		if err != nil {
			return nil, errSaveFailed(err)
		}
		return &tunnelOutput{Body: tunnelStateFromView(s.svc.View(tunnel))}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-tunnel",
		Method:        http.MethodDelete,
		Path:          "/api/tunnels/{id}",
		Summary:       "Stop and remove a saved connection",
		Description:   "Removing an id that is not saved is not an error.",
		Tags:          []string{tagTunnels},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *tunnelIDInput) (*emptyOutput, error) {
		if err := s.svc.DeleteTunnel(in.ID); err != nil {
			return nil, errSaveFailed(err)
		}
		return &emptyOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "start-tunnel",
		Method:      http.MethodPost,
		Path:        "/api/tunnels/{id}/start",
		Summary:     "Start a connection",
		Description: "When the provider has to be downloaded first, state is installing and the " +
			"tunnel starts once the download finishes. Progress and every later state arrive on " +
			"/api/events.",
		Tags: []string{tagTunnels},
	}, func(ctx context.Context, in *tunnelIDInput) (*tunnelOutput, error) {
		installing, err := s.svc.StartTunnel(in.ID)
		if errors.Is(err, service.ErrTunnelNotFound) {
			return nil, errTunnelNotFound()
		}
		if err != nil {
			return nil, apiError(http.StatusInternalServerError, "start_failed", err.Error())
		}

		state, ok := s.tunnelState(in.ID)
		if !ok {
			return nil, errTunnelNotFound()
		}
		if installing {
			state = TunnelState{
				Type:     eventTunnelState,
				ID:       state.ID,
				Name:     state.Name,
				Provider: state.Provider,
				Port:     state.Port,
				State:    string(core.TunnelStateInstall),
			}
		}
		return &tunnelOutput{Body: state}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "stop-tunnel",
		Method:      http.MethodPost,
		Path:        "/api/tunnels/{id}/stop",
		Summary:     "Stop a connection",
		Description: "Answers with the connection's state, or an empty body when it is not saved.",
		Tags:        []string{tagTunnels},
	}, func(ctx context.Context, in *tunnelIDInput) (*optionalTunnelOutput, error) {
		if err := s.svc.StopTunnel(in.ID); err != nil {
			return nil, apiError(http.StatusInternalServerError, "stop_failed", err.Error())
		}
		state, ok := s.tunnelState(in.ID)
		if !ok {
			return &optionalTunnelOutput{}, nil
		}
		return &optionalTunnelOutput{Body: &state}, nil
	})
}
