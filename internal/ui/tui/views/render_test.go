package views

import "testing"

func sampleItems() []TunnelViewData {
	return []TunnelViewData{
		{
			Name:        "Foundry VTT (Default)",
			Provider:    "cloudflared",
			LocalPort:   30000,
			StatusState: 3,
			PublicURL:   "https://random-words-here.trycloudflare.com",
		},
		{
			Name:        "Configuración de partida",
			Provider:    "pinggy",
			LocalPort:   30001,
			StatusState: 4,
			ErrorMsg:    "connection refused",
		},
	}
}

func TestListViewRendersAtAnyWidth(t *testing.T) {
	for width := 0; width <= 140; width++ {
		v := NewListView()
		v.Width = width
		v.Height = 24
		v.Items = sampleItems()
		v.Dashboard = "http://localhost:40500"
		v.Sessions = 2
		v.Message = "URL copied"
		v.UpdateBadge = "update v0.11.0 available"

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ListView.Render() panicked at width %d: %v", width, r)
				}
			}()
			v.Render()
		}()
	}
}

func TestLogsViewRendersAtAnyWidth(t *testing.T) {
	for width := 0; width <= 140; width++ {
		v := NewLogsView()
		v.Width = width
		v.TunnelName = "Foundry VTT"
		v.Content = "2024-01-01 starting tunnel\n2024-01-01 connected\n"

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("LogsView.Render() panicked at width %d: %v", width, r)
				}
			}()
			v.Render()
		}()
	}
}

func TestEmptyStateRendersAtAnyWidth(t *testing.T) {
	for width := 0; width <= 140; width++ {
		v := NewEmptyState()
		v.Width = width
		v.Height = 24
		v.Dashboard = "http://localhost:40500"

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("EmptyState.Render() panicked at width %d: %v", width, r)
				}
			}()
			v.Render()
		}()
	}
}

func TestDownloadingViewRendersAtAnyWidth(t *testing.T) {
	for width := 0; width <= 140; width++ {
		v := NewDownloadingView()
		v.Width = width
		v.Name = "cloudflared"
		v.Percent = 42
		v.Current = 1024 * 1024
		v.Total = 8 * 1024 * 1024

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("DownloadingView.Render() panicked at width %d: %v", width, r)
				}
			}()
			v.Render("[####    ]")
		}()
	}
}
