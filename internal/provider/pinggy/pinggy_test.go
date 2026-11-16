package pinggy

import "testing"

func TestParseURL(t *testing.T) {
	p := &PinggyCliProvider{}

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "free link domain",
			line: "https://kolfa-190-62-84-79.run.pinggy-free.link",
			want: "https://kolfa-190-62-84-79.run.pinggy-free.link",
		},
		{
			name: "free net domain",
			line: "https://uqehf-190-62-84-79.free.pinggy.net",
			want: "https://uqehf-190-62-84-79.free.pinggy.net",
		},
		{
			name: "pro link domain",
			line: "https://abcd-1-2-3-4.a.pinggy.link",
			want: "https://abcd-1-2-3-4.a.pinggy.link",
		},
		{
			name: "upgrade notice points at the dashboard",
			line: "⚠ Warning: You are not authenticated. Upgrade to Pinggy Pro. https://dashboard.pinggy.io",
			want: "",
		},
		{
			name: "header line",
			line: "Remote URLs:",
			want: "",
		},
		{
			name: "stop hint",
			line: "Press Ctrl+C to stop the tunnel.",
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
