package cloudflared

import "testing"

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
	}{
		{"darwin", "arm64", "cloudflared-darwin-arm64.tgz"},
		{"darwin", "amd64", "cloudflared-darwin-amd64.tgz"},
		{"linux", "amd64", "cloudflared-linux-amd64"},
		{"linux", "arm64", "cloudflared-linux-arm64"},
		{"windows", "amd64", "cloudflared-windows-amd64.exe"},
	}

	for _, tt := range tests {
		got, err := assetName(tt.goos, tt.goarch)
		if err != nil || got != tt.want {
			t.Errorf("assetName(%s, %s) = %q, %v; want %q", tt.goos, tt.goarch, got, err, tt.want)
		}
	}

	if _, err := assetName("plan9", "amd64"); err == nil {
		t.Error("assetName(plan9) returned no error")
	}
}
