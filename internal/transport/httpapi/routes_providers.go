package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/service"
)

// ProviderInfo is one tunnel provider ftm offers.
type ProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type providerListOutput struct {
	Body []ProviderInfo
}

type installedInput struct {
	Refresh string `query:"refresh" doc:"1 skips the 6-hour cache of the latest releases"`
}

type installedOutput struct {
	Body struct {
		Providers []ProviderBinaryRow `json:"providers"`
	}
}

type providerUpdateInput struct {
	ID string `path:"id" doc:"Provider id"`
}

type providerUpdateOutput struct {
	Body struct {
		OK       bool   `json:"ok"`
		Provider string `json:"provider"`
		State    string `json:"state" enum:"updating"`
	}
}

func (s *Server) registerProviders(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-providers",
		Method:      http.MethodGet,
		Path:        "/api/providers",
		Summary:     "List the tunnel providers",
		Tags:        []string{tagProviders},
	}, func(ctx context.Context, _ *struct{}) (*providerListOutput, error) {
		all := config.AllProviders()
		out := &providerListOutput{Body: make([]ProviderInfo, 0, len(all))}
		for _, p := range all {
			out.Body = append(out.Body, ProviderInfo{ID: string(p), Name: i18n.ProviderText(string(p))})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-installed-providers",
		Method:      http.MethodGet,
		Path:        "/api/providers/installed",
		Summary:     "Report every provider binary and whether it can be updated",
		Description: "Rows come in provider order. The first call after startup can take a few " +
			"seconds; later calls are cached.",
		Tags: []string{tagProviders},
	}, func(ctx context.Context, in *installedInput) (*installedOutput, error) {
		refresh := in.Refresh == "1" || in.Refresh == "true"
		report := s.svc.ProviderBinaries(ctx, refresh)

		out := &installedOutput{}
		out.Body.Providers = make([]ProviderBinaryRow, 0, len(report))
		for _, b := range report {
			out.Body.Providers = append(out.Body.Providers, providerBinaryRow(b))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-provider",
		Method:      http.MethodPost,
		Path:        "/api/providers/{id}/update",
		Summary:     "Update or install ftm's copy of a provider binary",
		Description: "Runs in the background: progress arrives as install events and the result " +
			"once as a provider_updated event. Running tunnels on that provider are stopped " +
			"before the swap and started again after.",
		Tags:          []string{tagProviders},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *providerUpdateInput) (*providerUpdateOutput, error) {
		err := s.svc.UpdateProvider(ctx, config.Provider(in.ID))
		switch {
		case errors.Is(err, service.ErrUnknownProvider):
			return nil, apiError(http.StatusNotFound, "unknown_provider", "unknown provider")
		case errors.Is(err, service.ErrNotUpdatable):
			return nil, apiError(http.StatusBadRequest, "not_updatable", err.Error())
		case errors.Is(err, service.ErrExternalBinary):
			return nil, apiError(http.StatusConflict, "external_binary", err.Error())
		case errors.Is(err, service.ErrUpdateInProgress):
			return nil, apiError(http.StatusConflict, "update_in_progress", err.Error())
		case err != nil:
			return nil, apiError(http.StatusInternalServerError, "update_failed", err.Error())
		}

		out := &providerUpdateOutput{}
		out.Body.OK = true
		out.Body.Provider = in.ID
		out.Body.State = "updating"
		return out, nil
	})
}
