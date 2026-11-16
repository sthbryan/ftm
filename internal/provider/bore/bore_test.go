package bore

import "testing"

func TestParseURL(t *testing.T) {
	p := &BoreProvider{}

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "listening line",
			line: "2026-08-03T20:11:02Z  INFO bore_cli::client: listening at bore.pub:41234",
			want: "http://bore.pub:41234",
		},
		{
			name: "connection line without a port",
			line: "2026-08-03T20:11:02Z  INFO bore_cli::client: connected to server host=bore.pub",
			want: "",
		},
		{
			name: "unrelated line",
			line: "2026-08-03T20:11:02Z  INFO bore_cli::client: new connection",
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
