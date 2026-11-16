package httpapi

import (
	"context"
	"errors"
	"github.com/sthbryan/ftm/internal/service"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sthbryan/ftm/internal/updater"
	"github.com/sthbryan/ftm/internal/version"
)

type statusOutput struct {
	Body struct {
		Port      int    `json:"port"`
		Version   string `json:"version"`
		NativePiP bool   `json:"nativePip" doc:"Whether the desktop shell can open a floating panel"`
	}
}

// UpdateInfo is what the last ftm update check found.
type UpdateInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Tag        string `json:"tag"`
	AssetName  string `json:"assetName"`
	ReleaseURL string `json:"releaseUrl"`
	HasUpdate  bool   `json:"hasUpdate"`
	Method     string `json:"method" enum:"self,store,homebrew,download" doc:"How this installation updates; only self can apply an update from here"`
}

type updateInfoOutput struct {
	Body UpdateInfo
}

type applyUpdateInput struct {
	Body *struct {
		_ struct{} `json:"-" additionalProperties:"true"`
	} `required:"false"`
}

type okOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type detectPortOutput struct {
	Body struct {
		Ports     []int                   `json:"ports" doc:"Checked ports something listens on"`
		Scanned   []int                   `json:"scanned" doc:"Every port checked: the usual Foundry ports plus the one asked for"`
		Foundry   []int                   `json:"foundry" doc:"Busy ports where Foundry VTT answers"`
		Worlds    []service.FoundryStatus `json:"worlds" doc:"What each Foundry says on /api/status: world, system, users"`
		Suggested int                     `json:"suggested"`
	}
}

type detectPortInput struct {
	Port int `query:"port" required:"false" minimum:"0" maximum:"65535" doc:"Also check this port, e.g. a connection's own"`
}

type pipInput struct {
	Body struct {
		_  struct{} `json:"-" additionalProperties:"true"`
		ID string   `json:"id" required:"false" doc:"Tunnel id"`
	}
}

func (s *Server) registerApp(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-status",
		Method:      http.MethodGet,
		Path:        "/api/status",
		Summary:     "Server port, version and desktop features",
		Tags:        []string{tagApp},
	}, func(ctx context.Context, _ *struct{}) (*statusOutput, error) {
		out := &statusOutput{}
		out.Body.Port = s.Port()
		out.Body.Version = version.Version
		out.Body.NativePiP = s.HasPiP()
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-update",
		Method:      http.MethodGet,
		Path:        "/api/update",
		Summary:     "The last ftm update check",
		Description: "The check runs in the background at start and every 6 hours; until it " +
			"answers, latest is empty and hasUpdate false.",
		Tags: []string{tagApp},
	}, func(ctx context.Context, _ *struct{}) (*updateInfoOutput, error) {
		out := &updateInfoOutput{Body: UpdateInfo{Current: version.Version, Method: string(updater.DetectMethod())}}
		if info := s.svc.UpdateInfo(); info != nil {
			out.Body.Latest = info.LatestVersion
			out.Body.Tag = info.Tag
			out.Body.AssetName = info.AssetName
			out.Body.ReleaseURL = info.ReleaseURL
			out.Body.HasUpdate = info.HasUpdate
			out.Body.Method = string(info.Method)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "apply-update",
		Method:      http.MethodPost,
		Path:        "/api/update",
		Summary:     "Install the update the last check found and restart",
		Description: "After answering, ftm stops its tunnels, shuts down through its normal path " +
			"and starts the new version.",
		Tags: []string{tagApp},
	}, func(ctx context.Context, _ *applyUpdateInput) (*okOutput, error) {
		info := s.svc.UpdateInfo()
		if info == nil || !info.HasUpdate {
			return nil, apiError(http.StatusBadRequest, "no_update", "no update available")
		}
		if err := s.svc.ApplyUpdate(); err != nil {
			if errors.Is(err, updater.ErrNotSelfUpdatable) {
				return nil, apiError(http.StatusConflict, "not_self_updatable", err.Error())
			}
			return nil, apiError(http.StatusInternalServerError, "update_failed", err.Error())
		}

		// The owner shuts down gracefully, which lets this response finish.
		s.svc.RequestRestart()

		out := &okOutput{}
		out.Body.OK = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "detect-port",
		Method:      http.MethodGet,
		Path:        "/api/detect-port",
		Summary:     "Find a running Foundry on its usual ports, or on the one asked for",
		Tags:        []string{tagApp},
	}, func(ctx context.Context, in *detectPortInput) (*detectPortOutput, error) {
		out := &detectPortOutput{}
		out.Body.Ports, out.Body.Scanned, out.Body.Suggested = s.svc.DetectFoundryPorts(in.Port)
		out.Body.Worlds = s.svc.FoundryOn(ctx, out.Body.Ports)
		out.Body.Foundry = make([]int, 0, len(out.Body.Worlds))
		for _, status := range out.Body.Worlds {
			out.Body.Foundry = append(out.Body.Foundry, status.Port)
		}
		if len(out.Body.Foundry) > 0 {
			out.Body.Suggested = out.Body.Foundry[0]
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "open-pip",
		Method:        http.MethodPost,
		Path:          "/api/pip",
		Summary:       "Open the desktop floating panel for a tunnel",
		Tags:          []string{tagApp},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *pipInput) (*emptyOutput, error) {
		if in.Body.ID == "" {
			return nil, apiError(http.StatusBadRequest, "missing_id", "id is required")
		}
		if _, ok := s.svc.Tunnel(in.Body.ID); !ok {
			return nil, errTunnelNotFound()
		}
		if err := s.OpenPiP(in.Body.ID); err != nil {
			if errors.Is(err, ErrNoPiP) {
				return nil, apiError(http.StatusNotImplemented, "pip_unavailable", err.Error())
			}
			return nil, apiError(http.StatusInternalServerError, "pip_failed", err.Error())
		}
		return &emptyOutput{}, nil
	})
}
