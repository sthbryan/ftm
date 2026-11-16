package tunnelmole

import "testing"

func TestParseURL(t *testing.T) {
	p := &TunnelmoleProvider{}

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "https forwarding line",
			line: "https://abcdef-ip-190-62-84-79.tunnelmole.net ⟶ http://localhost:30000",
			want: "https://abcdef-ip-190-62-84-79.tunnelmole.net",
		},
		{
			name: "http forwarding line",
			line: "http://abcdef-ip-190-62-84-79.tunnelmole.net ⟶ http://localhost:30000",
			want: "http://abcdef-ip-190-62-84-79.tunnelmole.net",
		},
		{
			name: "dashboard notice",
			line: "Your tunnel is listed at https://dashboard.tunnelmole.com",
			want: "",
		},
		{
			name: "install notice",
			line: "A new version is available at https://tunnelmole.com",
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
