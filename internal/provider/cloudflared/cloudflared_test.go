package cloudflared

import "testing"

func TestParseURL(t *testing.T) {
	p := &CloudflaredProvider{}

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "quick tunnel banner",
			line: "2026-08-03T20:11:02Z INF |  https://ntsc-project-router-giving.trycloudflare.com                    |",
			want: "https://ntsc-project-router-giving.trycloudflare.com",
		},
		{
			name: "failed request line is not a url announcement",
			line: `2026-08-03T20:12:10Z ERR Request failed error="dial tcp [::1]:30000: connect: connection refused" connIndex=0 dest=https://ntsc-project-router-giving.trycloudflare.com/favicon.ico ip=198.41.200.113 type=http`,
			want: "",
		},
		{
			name: "registration line without a tunnel host",
			line: "2026-08-03T20:11:01Z INF Requesting new quick Tunnel on trycloudflare.com...",
			want: "",
		},
		{
			name: "unrelated host",
			line: "2026-08-03T20:11:01Z INF see https://developers.cloudflare.com/argo-tunnel",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.ParseURL(tt.line); got != tt.want {
				t.Errorf("ParseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
