package ssh

import "testing"

func TestParseURLLocalhostRun(t *testing.T) {
	p := NewLocalhostRun()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "tunneled line",
			line: "abc123def.lhr.life tunneled with tls termination, https://abc123def.lhr.life",
			want: "https://abc123def.lhr.life",
		},
		{
			name: "authentication notice",
			line: "===============================================================================",
			want: "",
		},
		{
			name: "admin link is not the tunnel",
			line: "Follow your tunnel on https://admin.localhost.run/",
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

func TestParseURLServeo(t *testing.T) {
	p := NewServeo()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "forwarding line",
			line: "Forwarding HTTP traffic from https://abcdef.serveo.net",
			want: "https://abcdef.serveo.net",
		},
		{
			name: "user content host",
			line: "Forwarding HTTP traffic from https://abcdef.serveousercontent.com",
			want: "https://abcdef.serveousercontent.com",
		},
		{
			name: "press ctrl c",
			line: "Press g to start a GUI session and ctrl-c to quit.",
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
