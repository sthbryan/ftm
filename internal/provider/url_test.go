package provider

import "testing"

func TestExtractURL(t *testing.T) {
	always := func(string) bool { return true }

	tests := []struct {
		name  string
		line  string
		allow func(string) bool
		want  string
	}{
		{
			name: "bare url",
			line: "https://kolfa-190-62-84-79.run.pinggy-free.link",
			want: "https://kolfa-190-62-84-79.run.pinggy-free.link",
		},
		{
			name: "url inside a boxed banner",
			line: "|  https://ntsc-project-router-giving.trycloudflare.com     |",
			want: "https://ntsc-project-router-giving.trycloudflare.com",
		},
		{
			name: "trailing sentence punctuation is dropped",
			line: "Your site is available at https://foo.tunnelmole.net.",
			want: "https://foo.tunnelmole.net",
		},
		{
			name: "ansi colours are stripped",
			line: "\x1b[32mhttps://abc.lhr.life\x1b[0m",
			want: "https://abc.lhr.life",
		},
		{
			name: "quoted url",
			line: `dest="https://abc.trycloudflare.com/favicon.ico"`,
			want: "https://abc.trycloudflare.com/favicon.ico",
		},
		{
			name:  "host filter skips to the next url",
			line:  "visit https://dashboard.pinggy.io or https://abc.free.pinggy.net",
			allow: func(host string) bool { return host != "dashboard.pinggy.io" },
			want:  "https://abc.free.pinggy.net",
		},
		{
			name: "no url",
			line: "Press Ctrl+C to stop the tunnel.",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allow := tt.allow
			if allow == nil {
				allow = always
			}

			if got := ExtractURL(tt.line, allow); got != tt.want {
				t.Errorf("ExtractURL(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsSubdomainOf(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"abc.trycloudflare.com", true},
		{"a.b.trycloudflare.com", true},
		{"trycloudflare.com", false},
		{"trycloudflare.com.evil.test", false},
		{"nottrycloudflare.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := IsSubdomainOf(tt.host, "trycloudflare.com"); got != tt.want {
				t.Errorf("IsSubdomainOf(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
